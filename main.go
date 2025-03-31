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
var thedocument Document = Document{}
var themodel model = model{}

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
	for i, textInput := range themodel.textInputs {
		if i == themodel.cursor {
			textInput.TextModel.Focus()
			textInput.TextModel, _ = textInput.TextModel.Update(msg)
			themodel.textInputs[i] = textInput
		} else {
			textInput.TextModel.Blur()
			themodel.textInputs[i] = textInput
		}
	}
}

func updateTextSpecialElements() {
	for i, element := range thedocument.Elements {
		if element.Class == "input" {
			if isFocused(element.Id) {
				for _, textInput := range themodel.textInputs {
					if textInput.Id == element.Id {
						element.Text = textInput.TextModel.Value()
						thedocument.Elements[i] = element
					}
				}
			}
		}
	}
}

func isFocused(Id string) bool {
	for _, input := range themodel.textInputs {
		if input.Id == Id {
			return input.TextModel.Focused()
		}
	}
	return false
}

func callAllSpecialCallables() {
	for _, element := range thedocument.Elements {
		if element.Class == "input" && isFocused(element.Id) {
			if element.Callable != "" {
				state.Global(element.Callable)
				state.PushGoFunction(func(p *lua.State) int {
					err, _ := p.ToInteger(1)
					log(string(err))
					return 0
				})
				state.ProtectedCall(0, 0, 1)
				updateTextSpecialElements()
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
			themodel.cursor++
		}
	}
	if themodel.cursor > len(themodel.textInputs)-1 {
		themodel.cursor = 0
	}
	updateAllSpecialElements(msg)
	updateTextSpecialElements()
	callAllSpecialCallables()
	return m, nil
}

func (m model) View() string {
	text := applyBold(thedocument.Title)
	text += "\n"
	for _, element := range thedocument.Elements {
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
	fmt.Println("\n\n" + mystr)
	return 0
}

func applyBold(text string) string {
	return fmt.Sprint("\x1B[1m" + text + "\x1B[0m")
}
func createInitialSpecialElements() {
	found_special := false
	for _, element := range thedocument.Elements {
		if element.Class == "input" {
			textInput := textinput.New()
			textInput.Placeholder = element.Placeholder
			textInput.CharLimit = 255
			textInput.Width = 32
			if !found_special {
				textInput.Focus()
				found_special = true
			}
			themodel.textInputs = append(themodel.textInputs, TextInput{textInput, element.Id})

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
	thedocument.Elements = append(thedocument.Elements, myelement)
	return 0
}

func get_by_Id(p *lua.State) int {
	Id, _ := p.ToString(1)
	for _, element := range thedocument.Elements {
		if element.Id == Id {
			p.PushUserData(element)
		}
	}
	return 1
}

func set_by_Id(p *lua.State) int {
	Id, _ := p.ToString(1)
	AAAA := p.ToUserData(2)
	newElement := AAAA.(Element)
	for i, element := range thedocument.Elements {
		if element.Id == Id {
			thedocument.Elements[i] = newElement
		}
	}
	return 0
}

func param_of_element(p *lua.State) int {
	param, _ := p.ToString(1)
	myElement := p.ToUserData(2)
	element := myElement.(Element)

	var toPush string
	if param == "text" {
		toPush = element.Text
	} else if param == "class" {
		toPush = element.Class
	} else if param == "callable" {
		toPush = element.Callable
	} else if param == "placeholder" {
		toPush = element.Placeholder
	} else {
		toPush = "No Param Found"
	}
	p.PushString(toPush)
	return 1
}

func log(text string) {
	file, _ := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	file.WriteString(text + "\n")
}

func lua_log(p *lua.State) int {
	text, _ := p.ToString(-1)
	log(text)
	return 0
}

func main() {
	XML := `
  <document>
<title>Hello Title</title>
<elements>
  <element class="text">Hello from XML! Enter your name:</element>
  <element class="input" id="name" callable="name_handler" placeholder="John Doe"></element>
  <element class="text" id="myid">Test</element>
</elements>
<script>
add_element(new_element("text", "Hello, from Lua!"))
--add_element(new_element("input", "", "randomID", "callable1", "Ok"))
function name_handler()
    local text = param_of_element("text", get_by_Id("name"))
    local newElem = new_element("text", text) 
    set_by_Id("myid", newElem)
end
</script>
  </document>
`
	err := xml.Unmarshal([]byte(XML), &thedocument)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	lua.OpenLibraries(state)
	state.Register("log", lua_log)
	state.Register("add_element", add_element)
	state.Register("new_element", new_element)
	state.Register("get_by_Id", get_by_Id)
	state.Register("set_by_Id", set_by_Id)
	state.Register("param_of_element", param_of_element)
	err = lua.DoString(state, thedocument.Script)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	createInitialSpecialElements()
	program := tea.NewProgram(themodel)
	program.Run()
}
