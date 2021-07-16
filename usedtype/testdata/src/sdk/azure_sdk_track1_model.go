package sdk

// Hierarchy:
//
// 	Expected:												Actual:
//
//     BasicTop (Interface)                                        BasicMiddle (Interface)
//         ^                                                           ^
//         |                                                           |
//    +----+----+                                                  BasicTop (Interface)
//    |         |                                                      ^
//    A      BasicMiddle (Interface)                                   |
//              ^                                           +----------+----------+
//              |                                           |     |    |    |     |
//           +--+--+                                        A     B    C   Top  Middle
//           |     |
//           B     C

type BasicTop interface {
	AsTop() (*Top, bool)
	AsA() (*A, bool)
	AsBasicMiddle() (BasicMiddle, bool)
	AsMiddle() (*Middle, bool)
	AsB() (*B, bool)
	AsC() (*C, bool)
}

type BasicMiddle interface {
	AsMiddle() (*Middle, bool)
	AsB() (*B, bool)
	AsC() (*C, bool)
}

type Top struct{}

func (v Top) AsTop() (*Top, bool)                { return nil, false }
func (v Top) AsA() (*A, bool)                    { return nil, false }
func (v Top) AsBasicMiddle() (BasicMiddle, bool) { return nil, false }
func (v Top) AsMiddle() (*Middle, bool)          { return nil, false }
func (v Top) AsB() (*B, bool)                    { return nil, false }
func (v Top) AsC() (*C, bool)                    { return nil, false }

type A struct {
	Name string
}

func (v A) AsTop() (*Top, bool)                { return nil, false }
func (v A) AsA() (*A, bool)                    { return nil, false }
func (v A) AsBasicMiddle() (BasicMiddle, bool) { return nil, false }
func (v A) AsMiddle() (*Middle, bool)          { return nil, false }
func (v A) AsB() (*B, bool)                    { return nil, false }
func (v A) AsC() (*C, bool)                    { return nil, false }

type Middle struct{}

func (v Middle) AsTop() (*Top, bool)                { return nil, false }
func (v Middle) AsA() (*A, bool)                    { return nil, false }
func (v Middle) AsBasicMiddle() (BasicMiddle, bool) { return nil, false }
func (v Middle) AsMiddle() (*Middle, bool)          { return nil, false }
func (v Middle) AsB() (*B, bool)                    { return nil, false }
func (v Middle) AsC() (*C, bool)                    { return nil, false }

type B struct {
	Name string
}

func (v B) AsTop() (*Top, bool)                { return nil, false }
func (v B) AsA() (*A, bool)                    { return nil, false }
func (v B) AsBasicMiddle() (BasicMiddle, bool) { return nil, false }
func (v B) AsMiddle() (*Middle, bool)          { return nil, false }
func (v B) AsB() (*B, bool)                    { return nil, false }
func (v B) AsC() (*C, bool)                    { return nil, false }

type C struct {
	Name string
}

func (v C) AsTop() (*Top, bool)                { return nil, false }
func (v C) AsA() (*A, bool)                    { return nil, false }
func (v C) AsBasicMiddle() (BasicMiddle, bool) { return nil, false }
func (v C) AsMiddle() (*Middle, bool)          { return nil, false }
func (v C) AsB() (*B, bool)                    { return nil, false }
func (v C) AsC() (*C, bool)                    { return nil, false }
