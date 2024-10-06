package initropts

type Builder[K any] interface {
	Build() (K, error)
}
