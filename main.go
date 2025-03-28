package main

import (
	"encoding/xml"
	"fmt"
	lua "github.com/Shopify/go-lua"
	"os"
	//"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var mydocument Document = Document{}
var mymodel model = model{}

type Document struct {
	callables []func()
	Title     string    `xml:"title"`
	Elements  []Element `xml:"elements"`
	Script    string    `xml:"script"`
}

type Element struct {
	Class    string `xml:"class,attr"`
	Text     string `xml:"element"`
	Id       string `xml:"id,attr"`
	Callable string `xml:"callable,attr"`
}

type model struct {
	cursor []string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	text := applyBold(mydocument.Title)
	text += "\n"
	for _, element := range mydocument.Elements {
		text += element.Text + "\n"
	}
	return text
}

func Print(p *lua.State) int {
	mystr, _ := p.ToString(1)
	fmt.Println(mystr)
	return 0
}

func applyBold(text string) string {
	return fmt.Sprint("\x1B[1m" + text + "\x1B[0m")
}

func new_element(p *lua.State) int {
	class, _ := p.ToString(1)
	text, _ := p.ToString(2)
	id, _ := p.ToString(3)
	callable, _ := p.ToString(4)
	p.PushUserData(Element{class, text, id, callable})
	return 1
}

func add_element(p *lua.State) int {
	myInterfacedElement := p.ToUserData(1)
	myelement := myInterfacedElement.(Element)
	mydocument.Elements = append(mydocument.Elements, myelement)
	return 0
}
func main() {
	XML := `
  <document>
<title>Hello</title>
<elements>
  <element class="text" id="mytext" callable="nothing">Hello from XML!</element>
</elements>
<script>
add_element(new_element("text", "Hello, from Lua!", "", ""))
</script>
  </document>
`
	err := xml.Unmarshal([]byte(XML), &mydocument)
	if err != nil {
		fmt.Println(err)
	}
	state := lua.NewState()
	lua.OpenLibraries(state)
	state.Register("Print", Print)
	state.Register("add_element", add_element)
	state.Register("new_element", new_element)
	err = lua.DoString(state, mydocument.Script)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	program := tea.NewProgram(mymodel)
	program.Run()
}
