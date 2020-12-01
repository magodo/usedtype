package usedtype

import (
	"sort"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type ODUChainCluster map[ssa.Value]ODUChains

func (cluster ODUChainCluster) Pair() {
	// Provider chains always have pairing consumer chains, since now that they marked as provider chains,
	// it means they reach to in-boundary code that consume the value, i.e. the consumer chain exists.
	// On the other hand, some consumer chains might have no pairing provider chains, because they might be
	// waiting for const value, or out of boundary functions.
	// We will mark those consumer chains as consumer with no provider chains.
	providerCache := cluster.AllProviders()

	// Flatten all the chains
	var allChains []*ODUChain
	for _, chains := range cluster {
		for _, chain := range chains {
			allChains = append(allChains, &chain)
		}
	}

	// Iterate each chain and check on each consumer chain to see whether there is provider chain related.
	for _, chain := range allChains {
		if chain.state == chainOnHoldConsumer {
			lastInstr := chain.instrChain[len(chain.instrChain)-1]
			if _, ok := providerCache[lastInstr]; !ok {
				chain.state = chainEndNoProvider
			}
		}
	}

	// Only chains that have all marked to end can be used as provider.
	providerCache = cluster.AllProviders()
	for len(providerCache) != 0 {
		for v, chains := range cluster {
			var newChains ODUChains
			for _, chain := range chains {
				if chain.state != chainOnHoldConsumer {
					newChains = append(newChains, chain)
					continue
				}

				consumerChain := chain
				lastInstr := consumerChain.instrChain[len(consumerChain.instrChain)-1]
				providers, ok := providerCache[lastInstr]
				if !ok {
					// Current provider cache can not provide, waiting to be handled in the next loop
					newChains = append(newChains, consumerChain)
					continue
				}
				for _, provider := range providers {
					newChains = append(newChains, consumerChain.consume(cluster, provider.root)...)
				}
			}

			cluster[v] = newChains
		}

		// refresh providerCache
		providerCache = cluster.AllProviders()
	}
}

// AllProviders firstly find all the sets of chains that CAN provide something, then for each set of chains,
// find the real value provider, then set it into a map that is keyed by the consumer-provider instruction.
// Note that each consumer-provider instruction might be 1-N mapping, because (e.g.) there might be multiple return
// statements in a provider function.
func (cluster ODUChainCluster) AllProviders() map[ssa.Instruction][]ODUChain {
	providerCache := map[ssa.Instruction][]ODUChain{}
	for _, chains := range cluster {
		if !chains.CanProvide() {
			continue
		}
		for _, chain := range chains {
			if chain.state == chainOnHoldProvider {
				lastInstr := chain.instrChain[len(chain.instrChain)-1]
				providerCache[lastInstr] = append(providerCache[lastInstr], chain)
			}
		}
	}
	return providerCache
}

func (cluster ODUChainCluster) String() string {
	var keys comparableValues
	for k := range cluster {
		keys = append(keys, k)
	}
	sort.Sort(keys)

	out := make([]string, len(keys))
	for idx, k := range keys {
		chains := cluster[k]
		out[idx] = chains.String()
	}
	return strings.Join(out, "\n\n")
}

type comparableValues []ssa.Value

func (cvs comparableValues) Len() int {
	return len(cvs)
}

func (cvs comparableValues) Swap(i, j int) {
	cvs[i], cvs[j] = cvs[j], cvs[i]
}

func (cvs comparableValues) Less(i, j int) bool {
	return cvs[i].Pos() < cvs[j].Pos()
}
