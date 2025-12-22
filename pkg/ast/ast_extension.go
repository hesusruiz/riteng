package ast

import "fmt"

// IncludeNode represents <x-include>
type IncludeNode struct {
	BaseNode
	Href string
}

func (i *IncludeNode) TokenLiteral() string { return i.TokenLat }
func (i *IncludeNode) String() string { return fmt.Sprintf("<Include Href=%q />", i.Href) }
func (i *IncludeNode) blockNode() {}
