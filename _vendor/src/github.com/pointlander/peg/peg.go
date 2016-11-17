// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/pointlander/jetset"
)

const pegHeaderTemplate = `package {{.PackageName}}

import (
	{{range .Imports}}"{{.}}"
	{{end}}
)

const endSymbol rune = {{.EndSymbol}}

/* The rule types inferred from the grammar are below. */
type pegRule {{.PegRuleType}}

const (
	ruleUnknown pegRule = iota
	{{range .RuleNames}}rule{{.String}}
	{{end}}
)

var rul3s = [...]string {
	"Unknown",
	{{range .RuleNames}}"{{.String}}",
	{{end}}
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) Print(buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(string(([]rune(buffer)[node.begin:node.end]))))
			if node.up != nil {
				print(node.up, depth + 1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

type tokens32 struct {
	tree		[]token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2 * len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin: begin,
		end: end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type {{.StructName}} struct {
	{{.StructVariables}}
	Buffer		string
	buffer		[]rune
	rules		[{{.RulesCount}}]func() bool
	Parse		func(rule ...int) error
	Reset		func()
	Pretty 	bool
	tokens32
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int] textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

	search: for i, c := range buffer {
		if c == '\n' {line, symbol = line + 1, 0} else {symbol++}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {if i != positions[j] {continue search}}
			break search
		}
 	}

	return translations
}

type parseError struct {
	p *{{.StructName}}
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2 * len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p + 1
		positions[p], p = int(token.end), p + 1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
                         rul3s[token.pegRule],
                         translations[begin].line, translations[begin].symbol,
                         translations[end].line, translations[end].symbol,
                         strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *{{.StructName}}) PrintSyntaxTree() {
	p.tokens32.PrintSyntaxTree(p.Buffer)
}

{{if .HasActions}}
func (p *{{.StructName}}) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch (token.pegRule) {
		{{if .HasPush}}
		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])
		{{end}}
		{{range .Actions}}case ruleAction{{.GetId}}:
			{{.String}}
		{{end}}
		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}
{{end}}

func (p *{{.StructName}}) Init() {
	var (
		max token32
		position, tokenIndex uint32
		buffer []rune
	)
	p.Reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer) - 1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.Reset()

	_rules, tree := p.rules, tokens32{tree: make([]token32, math.MaxInt16)}
	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	{{if .HasDot}}
	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}
	{{end}}

	{{if .HasCharacter}}
	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/
	{{end}}

	{{if .HasString}}
	matchString := func(s string) bool {
		i := position
		for _, c := range s {
			if buffer[i] != c {
				return false
			}
			i++
		}
		position = i
		return true
	}
	{{end}}

	{{if .HasRange}}
	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/
	{{end}}

	_rules = [...]func() bool {
		nil,`

type Type uint8

const (
	TypeUnknown Type = iota
	TypeRule
	TypeName
	TypeDot
	TypeCharacter
	TypeRange
	TypeString
	TypePredicate
	TypeStateChange
	TypeCommit
	TypeAction
	TypePackage
	TypeImport
	TypeState
	TypeAlternate
	TypeUnorderedAlternate
	TypeSequence
	TypePeekFor
	TypePeekNot
	TypeQuery
	TypeStar
	TypePlus
	TypePeg
	TypePush
	TypeImplicitPush
	TypeNil
	TypeLast
)

var TypeMap = [...]string{
	"TypeUnknown",
	"TypeRule",
	"TypeName",
	"TypeDot",
	"TypeCharacter",
	"TypeRange",
	"TypeString",
	"TypePredicate",
	"TypeCommit",
	"TypeAction",
	"TypePackage",
	"TypeImport",
	"TypeState",
	"TypeAlternate",
	"TypeUnorderedAlternate",
	"TypeSequence",
	"TypePeekFor",
	"TypePeekNot",
	"TypeQuery",
	"TypeStar",
	"TypePlus",
	"TypePeg",
	"TypePush",
	"TypeImplicitPush",
	"TypeNil",
	"TypeLast"}

func (t Type) GetType() Type {
	return t
}

type Node interface {
	fmt.Stringer
	debug()

	Escaped() string
	SetString(s string)

	GetType() Type
	SetType(t Type)

	GetId() int
	SetId(id int)

	Init()
	Front() *node
	Next() *node
	PushFront(value *node)
	PopFront() *node
	PushBack(value *node)
	Len() int
	Copy() *node
	Slice() []*node
}

type node struct {
	Type
	string
	id int

	front  *node
	back   *node
	length int

	/* use hash table here instead of Copy? */
	next *node
}

func (n *node) String() string {
	return n.string
}

func (n *node) debug() {
	if len(n.string) == 1 {
		fmt.Printf("%v %v '%v' %d\n", n.id, TypeMap[n.Type], n.string, n.string[0])
	} else {
		fmt.Printf("%v %v '%v'\n", n.id, TypeMap[n.Type], n.string)
	}
}

func (n *node) Escaped() string {
	return escape(n.string)
}

func (n *node) SetString(s string) {
	n.string = s
}

func (n *node) SetType(t Type) {
	n.Type = t
}

func (n *node) GetId() int {
	return n.id
}

func (n *node) SetId(id int) {
	n.id = id
}

func (n *node) Init() {
	n.front = nil
	n.back = nil
	n.length = 0
}

func (n *node) Front() *node {
	return n.front
}

func (n *node) Next() *node {
	return n.next
}

func (n *node) PushFront(value *node) {
	if n.back == nil {
		n.back = value
	} else {
		value.next = n.front
	}
	n.front = value
	n.length++
}

func (n *node) PopFront() *node {
	front := n.front

	switch true {
	case front == nil:
		panic("tree is empty")
	case front == n.back:
		n.front, n.back = nil, nil
	default:
		n.front, front.next = front.next, nil
	}

	n.length--
	return front
}

func (n *node) PushBack(value *node) {
	if n.front == nil {
		n.front = value
	} else {
		n.back.next = value
	}
	n.back = value
	n.length++
}

func (n *node) Len() (c int) {
	return n.length
}

func (n *node) Copy() *node {
	return &node{Type: n.Type, string: n.string, id: n.id, front: n.front, back: n.back, length: n.length}
}

func (n *node) Slice() []*node {
	s := make([]*node, n.length)
	for element, i := n.Front(), 0; element != nil; element, i = element.Next(), i+1 {
		s[i] = element
	}
	return s
}

/* A tree data structure into which a PEG can be parsed. */
type Tree struct {
	Rules      map[string]Node
	rulesCount map[string]uint
	node
	inline, _switch bool

	RuleNames       []Node
	PackageName     string
	Imports         []string
	EndSymbol       rune
	PegRuleType     string
	StructName      string
	StructVariables string
	RulesCount      int
	Bits            int
	HasActions      bool
	Actions         []Node
	HasPush         bool
	HasCommit       bool
	HasDot          bool
	HasCharacter    bool
	HasString       bool
	HasRange        bool
}

func New(inline, _switch bool) *Tree {
	return &Tree{Rules: make(map[string]Node),
		rulesCount: make(map[string]uint),
		inline:     inline,
		_switch:    _switch}
}

func (t *Tree) AddRule(name string) {
	t.PushFront(&node{Type: TypeRule, string: name, id: t.RulesCount})
	t.RulesCount++
}

func (t *Tree) AddExpression() {
	expression := t.PopFront()
	rule := t.PopFront()
	rule.PushBack(expression)
	t.PushBack(rule)
}

func (t *Tree) AddName(text string) {
	t.PushFront(&node{Type: TypeName, string: text})
}

func (t *Tree) AddDot() { t.PushFront(&node{Type: TypeDot, string: "."}) }
func (t *Tree) AddCharacter(text string) {
	t.PushFront(&node{Type: TypeCharacter, string: text})
}
func (t *Tree) AddDoubleCharacter(text string) {
	t.PushFront(&node{Type: TypeCharacter, string: strings.ToLower(text)})
	t.PushFront(&node{Type: TypeCharacter, string: strings.ToUpper(text)})
	t.AddAlternate()
}
func (t *Tree) AddHexaCharacter(text string) {
	hexa, _ := strconv.ParseInt(text, 16, 32)
	t.PushFront(&node{Type: TypeCharacter, string: string(hexa)})
}
func (t *Tree) AddOctalCharacter(text string) {
	octal, _ := strconv.ParseInt(text, 8, 8)
	t.PushFront(&node{Type: TypeCharacter, string: string(octal)})
}
func (t *Tree) AddPredicate(text string)   { t.PushFront(&node{Type: TypePredicate, string: text}) }
func (t *Tree) AddStateChange(text string) { t.PushFront(&node{Type: TypeStateChange, string: text}) }
func (t *Tree) AddNil()                    { t.PushFront(&node{Type: TypeNil, string: "<nil>"}) }
func (t *Tree) AddAction(text string)      { t.PushFront(&node{Type: TypeAction, string: text}) }
func (t *Tree) AddPackage(text string)     { t.PushBack(&node{Type: TypePackage, string: text}) }
func (t *Tree) AddImport(text string)      { t.PushBack(&node{Type: TypeImport, string: text}) }
func (t *Tree) AddState(text string) {
	peg := t.PopFront()
	peg.PushBack(&node{Type: TypeState, string: text})
	t.PushBack(peg)
}

func (t *Tree) addList(listType Type) {
	a := t.PopFront()
	b := t.PopFront()
	var l *node
	if b.GetType() == listType {
		l = b
	} else {
		l = &node{Type: listType}
		l.PushBack(b)
	}
	l.PushBack(a)
	t.PushFront(l)
}
func (t *Tree) AddAlternate() { t.addList(TypeAlternate) }
func (t *Tree) AddSequence()  { t.addList(TypeSequence) }
func (t *Tree) AddRange()     { t.addList(TypeRange) }
func (t *Tree) AddDoubleRange() {
	a := t.PopFront()
	b := t.PopFront()

	t.AddCharacter(strings.ToLower(b.String()))
	t.AddCharacter(strings.ToLower(a.String()))
	t.addList(TypeRange)

	t.AddCharacter(strings.ToUpper(b.String()))
	t.AddCharacter(strings.ToUpper(a.String()))
	t.addList(TypeRange)

	t.AddAlternate()
}

func (t *Tree) addFix(fixType Type) {
	n := &node{Type: fixType}
	n.PushBack(t.PopFront())
	t.PushFront(n)
}
func (t *Tree) AddPeekFor() { t.addFix(TypePeekFor) }
func (t *Tree) AddPeekNot() { t.addFix(TypePeekNot) }
func (t *Tree) AddQuery()   { t.addFix(TypeQuery) }
func (t *Tree) AddStar()    { t.addFix(TypeStar) }
func (t *Tree) AddPlus()    { t.addFix(TypePlus) }
func (t *Tree) AddPush()    { t.addFix(TypePush) }

func (t *Tree) AddPeg(text string) { t.PushFront(&node{Type: TypePeg, string: text}) }

func join(tasks []func()) {
	length := len(tasks)
	done := make(chan int, length)
	for _, task := range tasks {
		go func(task func()) { task(); done <- 1 }(task)
	}
	for d := <-done; d < length; d += <-done {
	}
}

func escape(c string) string {
	switch c {
	case "'":
		return "\\'"
	case "\"":
		return "\""
	default:
		c = strconv.Quote(c)
		return c[1 : len(c)-1]
	}
}

func (t *Tree) Compile(file string, out io.Writer) {
	t.AddImport("fmt")
	t.AddImport("math")
	t.AddImport("sort")
	t.AddImport("strconv")
	t.EndSymbol = 0x110000
	t.RulesCount++

	counts := [TypeLast]uint{}
	{
		var rule *node
		var link func(node Node)
		link = func(n Node) {
			nodeType := n.GetType()
			id := counts[nodeType]
			counts[nodeType]++
			switch nodeType {
			case TypeAction:
				n.SetId(int(id))
				copy, name := n.Copy(), fmt.Sprintf("Action%v", id)
				t.Actions = append(t.Actions, copy)
				n.Init()
				n.SetType(TypeName)
				n.SetString(name)
				n.SetId(t.RulesCount)

				emptyRule := &node{Type: TypeRule, string: name, id: t.RulesCount}
				implicitPush := &node{Type: TypeImplicitPush}
				emptyRule.PushBack(implicitPush)
				implicitPush.PushBack(copy)
				implicitPush.PushBack(emptyRule.Copy())
				t.PushBack(emptyRule)
				t.RulesCount++

				t.Rules[name] = emptyRule
				t.RuleNames = append(t.RuleNames, emptyRule)
			case TypeName:
				name := n.String()
				if _, ok := t.Rules[name]; !ok {
					emptyRule := &node{Type: TypeRule, string: name, id: t.RulesCount}
					implicitPush := &node{Type: TypeImplicitPush}
					emptyRule.PushBack(implicitPush)
					implicitPush.PushBack(&node{Type: TypeNil, string: "<nil>"})
					implicitPush.PushBack(emptyRule.Copy())
					t.PushBack(emptyRule)
					t.RulesCount++

					t.Rules[name] = emptyRule
					t.RuleNames = append(t.RuleNames, emptyRule)
				}
			case TypePush:
				copy, name := rule.Copy(), "PegText"
				copy.SetString(name)
				if _, ok := t.Rules[name]; !ok {
					emptyRule := &node{Type: TypeRule, string: name, id: t.RulesCount}
					emptyRule.PushBack(&node{Type: TypeNil, string: "<nil>"})
					t.PushBack(emptyRule)
					t.RulesCount++

					t.Rules[name] = emptyRule
					t.RuleNames = append(t.RuleNames, emptyRule)
				}
				n.PushBack(copy)
				fallthrough
			case TypeImplicitPush:
				link(n.Front())
			case TypeRule, TypeAlternate, TypeUnorderedAlternate, TypeSequence,
				TypePeekFor, TypePeekNot, TypeQuery, TypeStar, TypePlus:
				for _, node := range n.Slice() {
					link(node)
				}
			}
		}
		/* first pass */
		for _, node := range t.Slice() {
			switch node.GetType() {
			case TypePackage:
				t.PackageName = node.String()
			case TypeImport:
				t.Imports = append(t.Imports, node.String())
			case TypePeg:
				t.StructName = node.String()
				t.StructVariables = node.Front().String()
			case TypeRule:
				if _, ok := t.Rules[node.String()]; !ok {
					expression := node.Front()
					copy := expression.Copy()
					expression.Init()
					expression.SetType(TypeImplicitPush)
					expression.PushBack(copy)
					expression.PushBack(node.Copy())

					t.Rules[node.String()] = node
					t.RuleNames = append(t.RuleNames, node)
				}
			}
		}
		/* second pass */
		for _, node := range t.Slice() {
			if node.GetType() == TypeRule {
				rule = node
				link(node)
			}
		}
	}

	join([]func(){
		func() {
			var countRules func(node Node)
			ruleReached := make([]bool, t.RulesCount)
			countRules = func(node Node) {
				switch node.GetType() {
				case TypeRule:
					name, id := node.String(), node.GetId()
					if count, ok := t.rulesCount[name]; ok {
						t.rulesCount[name] = count + 1
					} else {
						t.rulesCount[name] = 1
					}
					if ruleReached[id] {
						return
					}
					ruleReached[id] = true
					countRules(node.Front())
				case TypeName:
					countRules(t.Rules[node.String()])
				case TypeImplicitPush, TypePush:
					countRules(node.Front())
				case TypeAlternate, TypeUnorderedAlternate, TypeSequence,
					TypePeekFor, TypePeekNot, TypeQuery, TypeStar, TypePlus:
					for _, element := range node.Slice() {
						countRules(element)
					}
				}
			}
			for _, node := range t.Slice() {
				if node.GetType() == TypeRule {
					countRules(node)
					break
				}
			}
		},
		func() {
			var checkRecursion func(node Node) bool
			ruleReached := make([]bool, t.RulesCount)
			checkRecursion = func(node Node) bool {
				switch node.GetType() {
				case TypeRule:
					id := node.GetId()
					if ruleReached[id] {
						fmt.Fprintf(os.Stderr, "possible infinite left recursion in rule '%v'\n", node)
						return false
					}
					ruleReached[id] = true
					consumes := checkRecursion(node.Front())
					ruleReached[id] = false
					return consumes
				case TypeAlternate:
					for _, element := range node.Slice() {
						if !checkRecursion(element) {
							return false
						}
					}
					return true
				case TypeSequence:
					for _, element := range node.Slice() {
						if checkRecursion(element) {
							return true
						}
					}
				case TypeName:
					return checkRecursion(t.Rules[node.String()])
				case TypePlus, TypePush, TypeImplicitPush:
					return checkRecursion(node.Front())
				case TypeCharacter, TypeString:
					return len(node.String()) > 0
				case TypeDot, TypeRange:
					return true
				}
				return false
			}
			for _, node := range t.Slice() {
				if node.GetType() == TypeRule {
					checkRecursion(node)
				}
			}
		}})

	if t._switch {
		var optimizeAlternates func(node Node) (consumes bool, s jetset.Set)
		cache, firstPass := make([]struct {
			reached, consumes bool
			s                 jetset.Set
		}, t.RulesCount), true
		optimizeAlternates = func(n Node) (consumes bool, s jetset.Set) {
			/*n.debug()*/
			switch n.GetType() {
			case TypeRule:
				cache := &cache[n.GetId()]
				if cache.reached {
					consumes, s = cache.consumes, cache.s
					return
				}

				cache.reached = true
				consumes, s = optimizeAlternates(n.Front())
				cache.consumes, cache.s = consumes, s
			case TypeName:
				consumes, s = optimizeAlternates(t.Rules[n.String()])
			case TypeDot:
				consumes = true
				/* TypeDot set doesn't include the EndSymbol */
				s = s.Add(uint64(t.EndSymbol))
				s = s.Complement(uint64(t.EndSymbol))
			case TypeString, TypeCharacter:
				consumes = true
				s = s.Add(uint64([]rune(n.String())[0]))
			case TypeRange:
				consumes = true
				element := n.Front()
				lower := []rune(element.String())[0]
				element = element.Next()
				upper := []rune(element.String())[0]
				s = s.AddRange(uint64(lower), uint64(upper))
			case TypeAlternate:
				consumes = true
				mconsumes, properties, c :=
					consumes, make([]struct {
						intersects bool
						s          jetset.Set
					}, n.Len()), 0
				for _, element := range n.Slice() {
					mconsumes, properties[c].s = optimizeAlternates(element)
					consumes = consumes && mconsumes
					s = s.Union(properties[c].s)
					c++
				}

				if firstPass {
					break
				}

				intersections := 2
			compare:
				for ai, a := range properties[0 : len(properties)-1] {
					for _, b := range properties[ai+1:] {
						if a.s.Intersects(b.s) {
							intersections++
							properties[ai].intersects = true
							continue compare
						}
					}
				}
				if intersections >= len(properties) {
					break
				}

				c, unordered, ordered, max :=
					0, &node{Type: TypeUnorderedAlternate}, &node{Type: TypeAlternate}, 0
				for _, element := range n.Slice() {
					if properties[c].intersects {
						ordered.PushBack(element.Copy())
					} else {
						class := &node{Type: TypeUnorderedAlternate}
						for d := 0; d < 256; d++ {
							if properties[c].s.Has(uint64(d)) {
								class.PushBack(&node{Type: TypeCharacter, string: string(d)})
							}
						}

						sequence, predicate, length :=
							&node{Type: TypeSequence}, &node{Type: TypePeekFor}, properties[c].s.Len()
						if length == 0 {
							class.PushBack(&node{Type: TypeNil, string: "<nil>"})
						}
						predicate.PushBack(class)
						sequence.PushBack(predicate)
						sequence.PushBack(element.Copy())

						if element.GetType() == TypeNil {
							unordered.PushBack(sequence)
						} else if length > max {
							unordered.PushBack(sequence)
							max = length
						} else {
							unordered.PushFront(sequence)
						}
					}
					c++
				}
				n.Init()
				if ordered.Front() == nil {
					n.SetType(TypeUnorderedAlternate)
					for _, element := range unordered.Slice() {
						n.PushBack(element.Copy())
					}
				} else {
					for _, element := range ordered.Slice() {
						n.PushBack(element.Copy())
					}
					n.PushBack(unordered)
				}
			case TypeSequence:
				classes, elements :=
					make([]struct {
						s jetset.Set
					}, n.Len()), n.Slice()

				for c, element := range elements {
					consumes, classes[c].s = optimizeAlternates(element)
					if consumes {
						elements, classes = elements[c+1:], classes[:c+1]
						break
					}
				}

				for c := len(classes) - 1; c >= 0; c-- {
					s = s.Union(classes[c].s)
				}

				for _, element := range elements {
					optimizeAlternates(element)
				}
			case TypePeekNot, TypePeekFor:
				optimizeAlternates(n.Front())
			case TypeQuery, TypeStar:
				_, s = optimizeAlternates(n.Front())
			case TypePlus, TypePush, TypeImplicitPush:
				consumes, s = optimizeAlternates(n.Front())
			case TypeAction, TypeNil:
				//empty
			}
			return
		}
		for _, element := range t.Slice() {
			if element.GetType() == TypeRule {
				optimizeAlternates(element)
				break
			}
		}

		for i, _ := range cache {
			cache[i].reached = false
		}
		firstPass = false
		for _, element := range t.Slice() {
			if element.GetType() == TypeRule {
				optimizeAlternates(element)
				break
			}
		}
	}

	var buffer bytes.Buffer
	defer func() {
		fileSet := token.NewFileSet()
		code, error := parser.ParseFile(fileSet, file, &buffer, parser.ParseComments)
		if error != nil {
			buffer.WriteTo(out)
			fmt.Printf("%v: %v\n", file, error)
			return
		}
		formatter := printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}
		error = formatter.Fprint(out, fileSet, code)
		if error != nil {
			buffer.WriteTo(out)
			fmt.Printf("%v: %v\n", file, error)
			return
		}

	}()

	_print := func(format string, a ...interface{}) { fmt.Fprintf(&buffer, format, a...) }
	printSave := func(n uint) { _print("\n   position%d, tokenIndex%d := position, tokenIndex", n, n) }
	printRestore := func(n uint) { _print("\n   position, tokenIndex = position%d, tokenIndex%d", n, n) }
	printTemplate := func(s string) {
		if error := template.Must(template.New("peg").Parse(s)).Execute(&buffer, t); error != nil {
			panic(error)
		}
	}

	t.HasActions = counts[TypeAction] > 0
	t.HasPush = counts[TypePush] > 0
	t.HasCommit = counts[TypeCommit] > 0
	t.HasDot = counts[TypeDot] > 0
	t.HasCharacter = counts[TypeCharacter] > 0
	t.HasString = counts[TypeString] > 0
	t.HasRange = counts[TypeRange] > 0

	var printRule func(n Node)
	var compile func(expression Node, ko uint)
	var label uint
	labels := make(map[uint]bool)
	printBegin := func() { _print("\n   {") }
	printEnd := func() { _print("\n   }") }
	printLabel := func(n uint) {
		_print("\n")
		if labels[n] {
			_print("   l%d:\t", n)
		}
	}
	printJump := func(n uint) {
		_print("\n   goto l%d", n)
		labels[n] = true
	}
	printRule = func(n Node) {
		switch n.GetType() {
		case TypeRule:
			_print("%v <- ", n)
			printRule(n.Front())
		case TypeDot:
			_print(".")
		case TypeName:
			_print("%v", n)
		case TypeCharacter:
			_print("'%v'", escape(n.String()))
		case TypeString:
			s := escape(n.String())
			_print("'%v'", s[1:len(s)-1])
		case TypeRange:
			element := n.Front()
			lower := element
			element = element.Next()
			upper := element
			_print("[%v-%v]", escape(lower.String()), escape(upper.String()))
		case TypePredicate:
			_print("&{%v}", n)
		case TypeStateChange:
			_print("!{%v}", n)
		case TypeAction:
			_print("{%v}", n)
		case TypeCommit:
			_print("commit")
		case TypeAlternate:
			_print("(")
			elements := n.Slice()
			printRule(elements[0])
			for _, element := range elements[1:] {
				_print(" / ")
				printRule(element)
			}
			_print(")")
		case TypeUnorderedAlternate:
			_print("(")
			elements := n.Slice()
			printRule(elements[0])
			for _, element := range elements[1:] {
				_print(" | ")
				printRule(element)
			}
			_print(")")
		case TypeSequence:
			_print("(")
			elements := n.Slice()
			printRule(elements[0])
			for _, element := range elements[1:] {
				_print(" ")
				printRule(element)
			}
			_print(")")
		case TypePeekFor:
			_print("&")
			printRule(n.Front())
		case TypePeekNot:
			_print("!")
			printRule(n.Front())
		case TypeQuery:
			printRule(n.Front())
			_print("?")
		case TypeStar:
			printRule(n.Front())
			_print("*")
		case TypePlus:
			printRule(n.Front())
			_print("+")
		case TypePush, TypeImplicitPush:
			_print("<")
			printRule(n.Front())
			_print(">")
		case TypeNil:
		default:
			fmt.Fprintf(os.Stderr, "illegal node type: %v\n", n.GetType())
		}
	}
	compile = func(n Node, ko uint) {
		switch n.GetType() {
		case TypeRule:
			fmt.Fprintf(os.Stderr, "internal error #1 (%v)\n", n)
		case TypeDot:
			_print("\n   if !matchDot() {")
			/*print("\n   if buffer[position] == endSymbol {")*/
			printJump(ko)
			/*print("}\nposition++")*/
			_print("}")
		case TypeName:
			name := n.String()
			rule := t.Rules[name]
			if t.inline && t.rulesCount[name] == 1 {
				compile(rule.Front(), ko)
				return
			}
			_print("\n   if !_rules[rule%v]() {", name /*rule.GetId()*/)
			printJump(ko)
			_print("}")
		case TypeRange:
			element := n.Front()
			lower := element
			element = element.Next()
			upper := element
			/*print("\n   if !matchRange('%v', '%v') {", escape(lower.String()), escape(upper.String()))*/
			_print("\n   if c := buffer[position]; c < rune('%v') || c > rune('%v') {", escape(lower.String()), escape(upper.String()))
			printJump(ko)
			_print("}\nposition++")
		case TypeCharacter:
			/*print("\n   if !matchChar('%v') {", escape(n.String()))*/
			_print("\n   if buffer[position] != rune('%v') {", escape(n.String()))
			printJump(ko)
			_print("}\nposition++")
		case TypeString:
			_print("\n   if !matchString(%v) {", strconv.Quote(n.String()))
			printJump(ko)
			_print("}")
		case TypePredicate:
			_print("\n   if !(%v) {", n)
			printJump(ko)
			_print("}")
		case TypeStateChange:
			_print("\n   %v", n)
		case TypeAction:
		case TypeCommit:
		case TypePush:
			fallthrough
		case TypeImplicitPush:
			ok, element := label, n.Front()
			label++
			nodeType, rule := element.GetType(), element.Next()
			printBegin()
			if nodeType == TypeAction {
				_print("\nadd(rule%v, position)", rule)
			} else {
				_print("\nposition%d := position", ok)
				compile(element, ko)
				_print("\nadd(rule%v, position%d)", rule, ok)
			}
			printEnd()
		case TypeAlternate:
			ok := label
			label++
			printBegin()
			elements := n.Slice()
			printSave(ok)
			for _, element := range elements[:len(elements)-1] {
				next := label
				label++
				compile(element, next)
				printJump(ok)
				printLabel(next)
				printRestore(ok)
			}
			compile(elements[len(elements)-1], ko)
			printEnd()
			printLabel(ok)
		case TypeUnorderedAlternate:
			done, ok := ko, label
			label++
			printBegin()
			_print("\n   switch buffer[position] {")
			elements := n.Slice()
			elements, last := elements[:len(elements)-1], elements[len(elements)-1].Front().Next()
			for _, element := range elements {
				sequence := element.Front()
				class := sequence.Front()
				sequence = sequence.Next()
				_print("\n   case")
				comma := false
				for _, character := range class.Slice() {
					if comma {
						_print(",")
					} else {
						comma = true
					}
					_print(" '%s'", escape(character.String()))
				}
				_print(":")
				compile(sequence, done)
				_print("\nbreak")
			}
			_print("\n   default:")
			compile(last, done)
			_print("\nbreak")
			_print("\n   }")
			printEnd()
			printLabel(ok)
		case TypeSequence:
			for _, element := range n.Slice() {
				compile(element, ko)
			}
		case TypePeekFor:
			ok := label
			label++
			printBegin()
			printSave(ok)
			compile(n.Front(), ko)
			printRestore(ok)
			printEnd()
		case TypePeekNot:
			ok := label
			label++
			printBegin()
			printSave(ok)
			compile(n.Front(), ok)
			printJump(ko)
			printLabel(ok)
			printRestore(ok)
			printEnd()
		case TypeQuery:
			qko := label
			label++
			qok := label
			label++
			printBegin()
			printSave(qko)
			compile(n.Front(), qko)
			printJump(qok)
			printLabel(qko)
			printRestore(qko)
			printEnd()
			printLabel(qok)
		case TypeStar:
			again := label
			label++
			out := label
			label++
			printLabel(again)
			printBegin()
			printSave(out)
			compile(n.Front(), out)
			printJump(again)
			printLabel(out)
			printRestore(out)
			printEnd()
		case TypePlus:
			again := label
			label++
			out := label
			label++
			compile(n.Front(), ko)
			printLabel(again)
			printBegin()
			printSave(out)
			compile(n.Front(), out)
			printJump(again)
			printLabel(out)
			printRestore(out)
			printEnd()
		case TypeNil:
		default:
			fmt.Fprintf(os.Stderr, "illegal node type: %v\n", n.GetType())
		}
	}

	/* lets figure out which jump labels are going to be used with this dry compile */
	printTemp, _print := _print, func(format string, a ...interface{}) {}
	for _, element := range t.Slice() {
		if element.GetType() != TypeRule {
			continue
		}
		expression := element.Front()
		if expression.GetType() == TypeNil {
			continue
		}
		ko := label
		label++
		if count, ok := t.rulesCount[element.String()]; !ok {
			continue
		} else if t.inline && count == 1 && ko != 0 {
			continue
		}
		compile(expression, ko)
	}
	_print, label = printTemp, 0

	/* now for the real compile pass */
	t.PegRuleType = "uint8"
	if length := int64(t.Len()); length > math.MaxUint32 {
		t.PegRuleType = "uint64"
	} else if length > math.MaxUint16 {
		t.PegRuleType = "uint32"
	} else if length > math.MaxUint8 {
		t.PegRuleType = "uint16"
	}
	printTemplate(pegHeaderTemplate)
	for _, element := range t.Slice() {
		if element.GetType() != TypeRule {
			continue
		}
		expression := element.Front()
		if implicit := expression.Front(); expression.GetType() == TypeNil || implicit.GetType() == TypeNil {
			if element.String() != "PegText" {
				fmt.Fprintf(os.Stderr, "rule '%v' used but not defined\n", element)
			}
			_print("\n  nil,")
			continue
		}
		ko := label
		label++
		_print("\n  /* %v ", element.GetId())
		printRule(element)
		_print(" */")
		if count, ok := t.rulesCount[element.String()]; !ok {
			fmt.Fprintf(os.Stderr, "rule '%v' defined but not used\n", element)
			_print("\n  nil,")
			continue
		} else if t.inline && count == 1 && ko != 0 {
			_print("\n  nil,")
			continue
		}
		_print("\n  func() bool {")
		if labels[ko] {
			printSave(ko)
		}
		compile(expression, ko)
		//print("\n  fmt.Printf(\"%v\\n\")", element.String())
		_print("\n   return true")
		if labels[ko] {
			printLabel(ko)
			printRestore(ko)
			_print("\n   return false")
		}
		_print("\n  },")
	}
	_print("\n }\n p.rules = _rules")
	_print("\n}\n")
}
