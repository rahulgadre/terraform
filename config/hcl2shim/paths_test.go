package hcl2shim

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestPathFromFlatmap(t *testing.T) {
	tests := []struct {
		Flatmap string
		Type    cty.Type
		Want    cty.Path
		WantErr string
	}{
		{
			Flatmap: "",
			Type:    cty.EmptyObject,
			Want:    nil,
		},
		{
			Flatmap: "attr",
			Type:    cty.EmptyObject,
			Want:    nil,
			WantErr: `attribute "attr" not found`,
		},
		{
			Flatmap: "foo",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.String,
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
			},
		},
		{
			Flatmap: "foo.#",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
			},
		},
		{
			Flatmap: "foo.1",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
				cty.IndexStep{Key: cty.NumberIntVal(1)},
			},
		},
		{
			Flatmap: "foo.1",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Tuple([]cty.Type{
					cty.String,
					cty.Bool,
				}),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
				cty.IndexStep{Key: cty.NumberIntVal(1)},
			},
		},
		{
			// a set index returns the set itself, since this being applied to
			// a diff and the set is changing.
			Flatmap: "foo.24534534",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.String),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
			},
		},
		{
			Flatmap: "foo.%",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Map(cty.String),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
			},
		},
		{
			Flatmap: "foo.baz",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Map(cty.Bool),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
				cty.IndexStep{Key: cty.StringVal("baz")},
			},
		},
		{
			Flatmap: "foo.bar.baz",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Map(
					cty.Map(cty.Bool),
				),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
				cty.IndexStep{Key: cty.StringVal("bar")},
				cty.IndexStep{Key: cty.StringVal("baz")},
			},
		},
		{
			Flatmap: "foo.bar.baz",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Map(
					cty.Object(map[string]cty.Type{
						"baz": cty.String,
					}),
				),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
				cty.IndexStep{Key: cty.StringVal("bar")},
				cty.GetAttrStep{Name: "baz"},
			},
		},
		{
			Flatmap: "foo.0.bar",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.Object(map[string]cty.Type{
					"bar": cty.String,
					"baz": cty.Bool,
				})),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
				cty.IndexStep{Key: cty.NumberIntVal(0)},
				cty.GetAttrStep{Name: "bar"},
			},
		},
		{
			Flatmap: "foo.34534534.baz",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.Object(map[string]cty.Type{
					"bar": cty.String,
					"baz": cty.Bool,
				})),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
			},
		},
		{
			Flatmap: "foo.bar.bang",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.String,
			}),
			WantErr: `invalid step "bar.bang"`,
		},
		{
			// there should not be any attribute names with dots
			Flatmap: "foo.bar.bang",
			Type: cty.Object(map[string]cty.Type{
				"foo.bar": cty.Map(cty.String),
			}),
			WantErr: `attribute "foo" not found`,
		},
		{
			// We can only handle key names with dots if the map elements are a
			// primitive type.
			Flatmap: "foo.bar.bop",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Map(cty.String),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
				cty.IndexStep{Key: cty.StringVal("bar.bop")},
			},
		},
		{
			Flatmap: "foo.bar.0.baz",
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Map(
					cty.List(
						cty.Map(cty.String),
					),
				),
			}),
			Want: cty.Path{
				cty.GetAttrStep{Name: "foo"},
				cty.IndexStep{Key: cty.StringVal("bar")},
				cty.IndexStep{Key: cty.NumberIntVal(0)},
				cty.IndexStep{Key: cty.StringVal("baz")},
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s as %#v", test.Flatmap, test.Type), func(t *testing.T) {
			got, err := requiresReplacePath(test.Flatmap, test.Type)

			if test.WantErr != "" {
				if err == nil {
					t.Fatalf("succeeded; want error: %s", test.WantErr)
				}
				if got, want := err.Error(), test.WantErr; !strings.Contains(got, want) {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			if !reflect.DeepEqual(got, test.Want) {
				t.Fatalf("incorrect path\ngot:  %#v\nwant: %#v\n", got, test.Want)
			}
		})
	}
}
