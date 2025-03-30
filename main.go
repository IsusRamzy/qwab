package main

import (
	"encoding/xml"
	"fmt"
	"os"

	lua "github.com/Shopify/go-lua"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var state *lua.State = lua.NewState()
var mydocument Document = Document{}
var mymodel model = model{}

type Document struct {
	Title    string    `xml:"title"`
	Elements []Element `xml:"elements>element"`
	Script   string    `xml:"script"`
}

type Element struct {
	Class       string `xml:"class,attr"`
	Text        string `xml:",chardata"`
	Id          string `xml:"id,attr"`
	Callable    string `xml:"callable,attr"`
	Placeholder string `xml:"placeholder,attr"`
}

type model struct {
	cursor     int
	textInputs []TextInput
}

type TextInput struct {
	TextModel textinput.Model
	Id        string
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func updateAllSpecialElements(msg tea.Msg) {
	for i, textInput := range mymodel.textInputs {
		if i == mymodel.cursor {
			textInput.TextModel.Focus()
			textInput.TextModel, _ = textInput.TextModel.Update(msg)
			mymodel.textInputs[i] = textInput
		} else {
			textInput.TextModel.Blur()
			mymodel.textInputs[i] = textInput
		}
	}
}

func isFocused(Id string) bool {
	for _, input := range mymodel.textInputs {
		if input.Id == Id {
			return input.TextModel.Focused()
		}
	}
	return false
}

func callAllSpecialCallables() {
	for _, element := range mydocument.Elements {
		if element.Class == "input" && isFocused(element.Id) {
			if element.Callable != "" {
				state.Global(element.Callable)
				state.ProtectedCall(0, 0, 0)
			}
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		} else if msg.Type == tea.KeyTab {
			mymodel.cursor++
		}
	}
	if mymodel.cursor > len(mymodel.textInputs)-1 {
		mymodel.cursor = 0
	}
	updateAllSpecialElements(msg)
	callAllSpecialCallables()
	return m, nil
}

func (m model) View() string {
	text := applyBold(mydocument.Title)
	text += "\n"
	for _, element := range mydocument.Elements {
		if element.Class == "text" {
			text += element.Text + "\n"
		} else if element.Class == "input" {
			for _, textInput := range m.textInputs {
				if textInput.Id == element.Id {
					text += textInput.TextModel.View()
					text += "\n"
					break
				}
			}
		}
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
func createInitialSpecialElements() {
	found_special := false
	for _, element := range mydocument.Elements {
		if element.Class == "input" {
			textInput := textinput.New()
			textInput.Placeholder = element.Placeholder
			textInput.CharLimit = 255
			textInput.Width = 32
			if !found_special {
				textInput.Focus()
				found_special = true
			}
			mymodel.textInputs = append(mymodel.textInputs, TextInput{textInput, element.Id})

		}
	}
}

func new_element(p *lua.State) int {
	class, _ := p.ToString(1)
	text, _ := p.ToString(2)
	id, _ := p.ToString(3)
	callable, _ := p.ToString(4)
	placeholder, _ := p.ToString(5)
	p.PushUserData(Element{class, text, id, callable, placeholder})
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
<title>Hello Title</title>
<elements>
  <element class="text">Hello from XML!</element>
  <element class="input" id="myinput" callable="XML" placeholder="input"></element>
</elements>
<script>
add_element(new_element("text", "Hello, from Lua!"))
add_element(new_element("input", "", "randomID", "callable1", "Ok"))
add_element(new_element("text", "Hello, second input from Lua!"))
add_element(new_element("input", "", "randomID2", "callable2", "Hi!"))
function callable1()
  add_element(new_element("text", "Called1"))
end
function callable2()
  add_element(new_element("text", "Called2"))
end

function XML()
  add_element(new_element("text", "XML"))
end
</script>
  </document>
`
	err := xml.Unmarshal([]byte(XML), &mydocument)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	lua.OpenLibraries(state)
	state.Register("Print", Print)
	state.Register("add_element", add_element)
	state.Register("new_element", new_element)
	err = lua.DoString(state, mydocument.Script)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	createInitialSpecialElements()
	program := tea.NewProgram(mymodel)
	program.Run()
}
