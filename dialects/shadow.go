package dialects

import "context"

type Shadowable interface {
	IsShadow(ctx context.Context) bool
}

type TrueShadow struct{}
type FalseShadow struct{}

func NewTrueShadow() Shadowable {
	return &TrueShadow{}
}
func NewFalseShadow() Shadowable {
	return &FalseShadow{}
}
func (t *TrueShadow) IsShadow(ctx context.Context) bool {
	return true
}
func (f *FalseShadow) IsShadow(ctx context.Context) bool {
	return false
}
