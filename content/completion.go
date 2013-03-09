package content

type Type struct {
	Name           FullyQualifiedName `protocol:"required" json:",omitempty"`
	Specialization []Type             `protocol:"optional" json:",omitempty"`
	Flags          Flags              `protocol:"optional" json:",omitempty"`
	Methods        []Method           `protocol:"optional" json:",omitempty"`
	Fields         []Field            `protocol:"optional" json:",omitempty"`
	Types          []Type             `protocol:"optional" json:",omitempty"`
	Extends        []Type             `protocol:"optional" json:",omitempty"`
	Implements     []Type             `protocol:"optional" json:",omitempty"`
}

const (
	FLAG_ACC_NOTSET = iota
	FLAG_ACC_PUBLIC
	FLAG_ACC_PRIVATE
	FLAG_ACC_PROTECTED
	FLAG_ACC_MASK = 0x3
	FLAG_ACC_BITS = 2
)
const (
	FLAG_STATIC = 1 << (FLAG_ACC_BITS + iota)
	FLAG_FINAL
	FLAG_CLASS
	FLAG_INTERFACE
)

type Flags uint32

type Variable struct {
	Name FullyQualifiedName `protocol:"required" json:",omitempty"`
	Type Type               `protocol:"required" json:",omitempty"`
}

type Field struct {
	Variable `protocol:"required"`
	Flags    Flags `protocol:"optional" json:",omitempty"`
}

type Method struct {
	Name           FullyQualifiedName `protocol:"required" json:",omitempty"`
	Flags          Flags              `protocol:"optional" json:",omitempty"`
	Returns        []Variable         `protocol:"optional" json:",omitempty"`
	Parameters     []Variable         `protocol:"optional" json:",omitempty"`
	Specialization []Type             `protocol:"optional" json:",omitempty"`
	Static         bool               `protocol:"optional"`
}

type CompletionResult struct {
	Types   []Type   `protocol:"optional" json:",omitempty"`
	Fields  []Field  `protocol:"optional" json:",omitempty"`
	Methods []Method `protocol:"optional" json:",omitempty"`
}
