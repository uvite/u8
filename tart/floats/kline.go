package floats

type Kline []any

func NewKline(a ...any) Kline {
	return Kline(a)
}

func (s *Kline) Push(v any) {
	*s = append(*s, v)
}

func (s *Kline) Update(v any) {
	*s = append(*s, v)
}

func (s *Kline) Pop(i int64) (v any) {
	v = (*s)[i]
	*s = append((*s)[:i], (*s)[i+1:]...)
	return v
}

func (s Kline) Tail(size int) Kline {
	length := len(s)
	if length <= size {
		win := make(Kline, length)
		copy(win, s)
		return win
	}

	win := make(Kline, size)
	copy(win, s[length-size:])
	return win
}

func (collection Kline) Reverse() Kline {
	length := len(collection)
	half := length / 2

	for i := 0; i < half; i = i + 1 {
		j := length - 1 - i
		collection[i], collection[j] = collection[j], collection[i]
	}

	return collection
}

func (s *Kline) Last() any {
	length := len(*s)
	if length > 0 {
		return (*s)[length-1]
	}
	return 0.0
}

func (s *Kline) Index(i int) any {
	length := len(*s)
	if length-i <= 0 || i < 0 {
		return 0.0
	}
	return (*s)[length-i-1]
}

func (s *Kline) Length() int {
	return len(*s)
}

func (s Kline) Addr() *Kline {
	return &s
}
func (s Kline) String(t Kline) any {

	return s.Index(0)

}
