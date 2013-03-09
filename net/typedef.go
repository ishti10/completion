package net

import (
	"errors"
	"github.com/quarnster/completion/content"
	"sort"
)

var (
	ErrInterface = errors.New("TypeDef is an interface, not a class")
)

type TypeDef struct {
	index TypeDefIndex
	row   TypeDefRow
}

type AbstractType interface {
	Name() string
	Namespace() string
}

func AbsoluteName(t AbstractType) string {
	n := t.Name()
	if ns := t.Namespace(); ns != "" {
		return ns + "." + n
	}
	return n
}

func ToContentType(t AbstractType) (t2 content.Type) {
	t2.Name.Relative = t.Name()
	if ns := t.Namespace(); ns != "" {
		t2.Name.Absolute = ns + "." + t2.Name.Relative
	}
	return
}

func (t *TypeDefRow) Name() string {
	return string(t.TypeName)
}

func (t *TypeDefRow) Namespace() string {
	return string(t.TypeNamespace)
}

func (t *TypeRefRow) Name() string {
	return string(t.TypeName)
}

func (t *TypeRefRow) Namespace() string {
	return string(t.TypeNamespace)
}

func (t *TypeSpecRow) Name() string {
	return string("spec")
}

func (t *TypeSpecRow) Namespace() string {
	return string("unknown")
}

func TypeDefFromIndex(index TypeDefIndex) (*TypeDef, error) {
	if tr, err := index.Data(); err != nil {
		return nil, err
	} else {
		return &TypeDef{index, *tr.(*TypeDefRow)}, nil
	}
}

func (td *TypeDef) initContentType(index TypeDefIndex, t *Type) (t2 content.Type) {
	switch t.TypeId {
	case ELEMENT_TYPE_GENERICINST:
		t2 = ToContentType(t.Type)
		for i := range t.Instance {
			t2.Specialization = append(t2.Specialization, td.initContentType(index, t.Instance[i]))
		}
	case ELEMENT_TYPE_VAR, ELEMENT_TYPE_MVAR:
		idx2 := td.Search(index, id_GenericParam, func(idx TableIndex) bool {
			if raw, err := idx.Data(); err == nil {
				gr := raw.(*GenericParamRow)
				return gr.Owner.Table() >= index.Table() && gr.Owner.Index() >= index.Index()
			}
			return false
		})
		if idx2.Table() != id_nullTable {
			ci := idx2.(*ConcreteTableIndex)
			ci.index += t.GenericNumber
			if raw, err := ci.Data(); err == nil {
				gr := raw.(*GenericParamRow)
				t2.Name.Relative = string(gr.Name)
			}
		}
	default:
		return ToContentType(t)
	}

	return
}

func (td *TypeDef) Search(index TypeDefIndex, tableId int, equal func(TableIndex) bool) TableIndex {
	mu := index.(*ConcreteTableIndex).metadataUtil
	table := mu.Tables[tableId]
	ci := ConcreteTableIndex{metadataUtil: mu, index: 0, table: tableId}
	idx := sort.Search(int(table.Rows), func(in int) bool {
		i := uint32(in)
		ci.index = i + 1
		return equal(&ci)
	})
	if uint32(idx) == table.Rows {
		return nil
	}
	ci.index = uint32(idx) + 1
	return &ci
}

func (td *TypeDef) Extends() (t []content.Type, err error) {
	if (td.row.Flags & TypeAttributes_ClassSemanticsMask) != TypeAttributes_Class {
		return nil, ErrInterface
	}
	if td.row.Extends.Index() != 0 {
		if raw, err := td.row.Extends.Data(); err != nil {
			return nil, err
		} else {
			t = append(t, ToContentType(raw.(AbstractType)))
		}
	}
	return
}

func (td *TypeDef) Implements() (interfaces []content.Type, err error) {
	mu := td.index.(*ConcreteTableIndex).metadataUtil
	table := mu.Tables[id_InterfaceImpl]
	rawidx := td.Search(td.index, id_InterfaceImpl, func(ti TableIndex) bool {
		if raw, err := ti.Data(); err == nil {
			c := raw.(*InterfaceImplRow)
			return c.Class.Table() >= td.index.Table() && c.Class.Index() >= td.index.Index()
		}
		return false
	})
	if rawidx == nil {
		return nil, nil
	}
	ci := rawidx.(*ConcreteTableIndex)
	for i := uint32(ci.index); i < table.Rows+1; i++ {
		ci.index = i
		if raw, err := ci.Data(); err != nil {
			return nil, err
		} else {
			c := raw.(*InterfaceImplRow)
			if c.Class.Index() != td.index.Index() {
				break
			}
			if raw, err := c.Interface.Data(); err != nil {
				return nil, err
			} else {
				interfaces = append(interfaces, ToContentType(raw.(AbstractType)))
			}
		}
	}
	return
}

func (td *TypeDef) ListRange(index uint32, table, memberTable int, getindex func(interface{}) uint32) (startRow, endRow uint32) {
	mu := td.index.(*ConcreteTableIndex).metadataUtil
	var (
		idx      = ConcreteTableIndex{mu, index, table}
		tableEnd = mu.Tables[memberTable].Rows + 1
	)
	if i, err := idx.Data(); err != nil {
		return 0, 0
	} else {
		startRow = getindex(i)
	}
	idx.index++
	if i, err := idx.Data(); err == nil {
		endRow = getindex(i)
	} else {
		endRow = tableEnd
	}
	if endRow < startRow {
		endRow = tableEnd
	}
	return
}

func (td *TypeDef) Fields() (fields []content.Field, err error) {
	var (
		mu               = td.index.(*ConcreteTableIndex).metadataUtil
		startRow, endRow = td.ListRange(td.index.Index(), id_TypeDef, id_Field, func(i interface{}) uint32 { return i.(*TypeDefRow).FieldList.Index() })
		idx              = ConcreteTableIndex{mu, startRow, id_Field}
	)
	for i := startRow; i < endRow; i++ {
		idx.index = i
		if rawfield, err := idx.Data(); err != nil {
			return nil, err
		} else {
			var (
				field = rawfield.(*FieldRow)
				f     content.Field
				dec   *SignatureDecoder
				sig   FieldSig
			)
			f.Name.Relative = string(field.Name)
			if dec, err = NewSignatureDecoder(field.Signature); err != nil {
				return nil, err
			} else if err = dec.Decode(&sig); err != nil {
				return nil, err
			} else {
				f.Type = td.initContentType(td.index, &sig.Type)
			}
			if field.Flags&FieldAttributes_Static != 0 {
				f.Flags |= content.FLAG_STATIC
			}
			if field.Flags&FieldAttributes_Public != 0 {
				f.Flags |= content.FLAG_ACC_PUBLIC
			} else if field.Flags&FieldAttributes_Private != 0 {
				f.Flags |= content.FLAG_ACC_PRIVATE
			} else if field.Flags&FieldAttributes_Family != 0 {
				f.Flags |= content.FLAG_ACC_PROTECTED
			}

			fields = append(fields, f)
		}
	}
	return fields, nil
}

func (td *TypeDef) Parameters(index MethodDefIndex) (params []content.Variable, err error) {
	var (
		mu               = td.index.(*ConcreteTableIndex).metadataUtil
		startRow, endRow = td.ListRange(index.Index(), id_MethodDef, id_Param, func(i interface{}) uint32 { return i.(*MethodDefRow).ParamList.Index() })
		idx              = ConcreteTableIndex{mu, startRow, id_Param}
	)
	for i := startRow; i < endRow; i++ {
		idx.index = i
		if rawparam, err := idx.Data(); err != nil {
			return nil, err
		} else {
			param := rawparam.(*ParamRow)
			var f content.Variable
			f.Name.Relative = string(param.Name)
			params = append(params, f)
		}
	}
	return params, nil
}

func (td *TypeDef) Methods() (methods []content.Method, err error) {
	var (
		mu               = td.index.(*ConcreteTableIndex).metadataUtil
		startRow, endRow = td.ListRange(td.index.Index(), id_TypeDef, id_MethodDef, func(i interface{}) uint32 { return i.(*TypeDefRow).MethodList.Index() })
		idx              = &ConcreteTableIndex{mu, startRow, id_MethodDef}
	)
	for i := startRow; i < endRow; i++ {
		idx.index = i
		if rawmethod, err := idx.Data(); err != nil {
			return nil, err
		} else {
			var (
				m      content.Method
				method = rawmethod.(*MethodDefRow)
				dec    *SignatureDecoder
				sig    MethodDefSig
			)
			m.Name.Relative = string(method.Name)
			if m.Parameters, err = td.Parameters(idx); err != nil {
				return nil, err
			}
			if dec, err = NewSignatureDecoder(method.Signature); err != nil {
				return nil, err
			} else if err = dec.Decode(&sig); err != nil {
				return nil, err
			} else {
				// TODO: need to figure out why this mismatch happens
				l := len(sig.Params)
				if l2 := len(m.Parameters); l2 < l {
					l = l2
				}
				for i := range sig.Params[:l] {
					m.Parameters[i].Type = td.initContentType(td.index, &sig.Params[i].Type)
				}
				if method.Flags&MethodAttributes_Final != 0 {
					m.Flags |= content.FLAG_FINAL
				}
				if method.Flags&MethodAttributes_Static != 0 {
					m.Flags |= content.FLAG_STATIC
				}
				if method.Flags&MethodAttributes_Public != 0 {
					m.Flags |= content.FLAG_ACC_PUBLIC
				} else if method.Flags&MethodAttributes_Private != 0 {
					m.Flags |= content.FLAG_ACC_PRIVATE
				} else if method.Flags&MethodAttributes_Family != 0 {
					m.Flags |= content.FLAG_ACC_PROTECTED
				}

				m.Returns = make([]content.Variable, 1)
				m.Returns[0].Type = td.initContentType(td.index, &sig.RetType.Type)
			}
			methods = append(methods, m)
		}
	}
	return methods, nil
}

func (td *TypeDef) ToContentType() (t content.Type, err error) {
	t = ToContentType(&td.row)
	switch t2 := td.row.Flags & TypeAttributes_ClassSemanticsMask; t2 {
	case TypeAttributes_Class:
		t.Flags |= content.FLAG_CLASS
	case TypeAttributes_Interface:
		t.Flags |= content.FLAG_INTERFACE
	}
	if td.row.Flags&TypeAttributes_Public != 0 {
		t.Flags |= content.FLAG_ACC_PUBLIC
	}

	if ext, err := td.Extends(); err != nil && err != ErrInterface {
		return content.Type{}, err
	} else {
		t.Extends = ext
	}
	if imp, err := td.Implements(); err != nil {
		return content.Type{}, err
	} else {
		t.Implements = imp
	}
	if f, err := td.Fields(); err != nil {
		return content.Type{}, err
	} else {
		t.Fields = f
	}
	if f, err := td.Methods(); err != nil {
		return content.Type{}, err
	} else {
		t.Methods = f
	}

	return
}
