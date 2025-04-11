package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

var state *lua.LState = lua.NewState()
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
	textAreas  []TextArea
}
type TextArea struct {
	TextModel textarea.Model
	Id        string
}
type TextInput struct {
	TextModel textinput.Model
	Id        string
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func findIndexOfElementById(Id string) int {
	for index, element := range thedocument.Elements {
		if element.Id == Id {
			return index
		}
	}
	return -404
}

func updateAllSpecialElements(msg tea.Msg) {
	for i, textInput := range themodel.textInputs {
		if findIndexOfElementById(textInput.Id) == themodel.cursor {
			textInput.TextModel.Focus()
			textInput.TextModel, _ = textInput.TextModel.Update(msg)
			themodel.textInputs[i] = textInput
		} else {
			textInput.TextModel.Blur()
			themodel.textInputs[i] = textInput
		}
	}
	for i, textArea := range themodel.textAreas {
		if findIndexOfElementById(textArea.Id) == themodel.cursor {
			textArea.TextModel.Focus()
			textArea.TextModel, _ = textArea.TextModel.Update(msg)
			themodel.textAreas[i] = textArea
		} else {
			textArea.TextModel.Blur()
			themodel.textAreas[i] = textArea
		}
	}
}

func updateTextSpecialElements() {
	for i, element := range thedocument.Elements {
		if element.Class == "input" || element.Class == "textarea" {
			if isFocused(element.Id) {
				for _, textInput := range themodel.textInputs {
					if textInput.Id == element.Id {
						element.Text = textInput.TextModel.Value()
						thedocument.Elements[i] = element
					}
				}
				for _, textInput := range themodel.textAreas {
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
	for _, input := range themodel.textAreas {
		if input.Id == Id {
			return input.TextModel.Focused()
		}
	}
	return false
}

func callAllSpecialCallables() {
	for _, element := range thedocument.Elements {
		if (element.Class == "textarea" || element.Class == "input") && isFocused(element.Id) {
			if element.Callable != "" {
				whatever := state.GetGlobal(element.Callable)
				state.Push(whatever)
				state.CallByParam(lua.P{
					Fn:   state.GetGlobal(element.Callable),
					NRet: 0,
				}, nil)
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
	if themodel.cursor > len(thedocument.Elements)-1 {
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
		} else if element.Class == "input" || element.Class == "textarea" {
			for _, textInput := range m.textInputs {
				if textInput.Id == element.Id {
					text += textInput.TextModel.View()
					text += "\n"
					break
				}
			}
			for _, textArea := range m.textAreas {
				if textArea.Id == element.Id {
					text += textArea.TextModel.View()
					text += "\n"
					break
				}
			}
		}
	}
	return text
}

func Print(p *lua.LState) int {
	mystr := p.ToString(1)
	fmt.Println("\n\n" + mystr)
	return 0
}

func applyBold(text string) string {
	return fmt.Sprint("\x1B[1m" + text + "\x1B[0m")
}
func createInitialSpecialElements() {
	foundInput := false
	foundTextarea := false
	for _, element := range thedocument.Elements {
		if element.Class == "input" {
			textInput := textinput.New()
			textInput.Placeholder = element.Placeholder
			textInput.CharLimit = 255
			textInput.Width = 32
			if !foundInput {
				textInput.Focus()
				foundInput = true
			}
			themodel.textInputs = append(themodel.textInputs, TextInput{textInput, element.Id})
		} else if element.Class == "textarea" {
			textArea := textarea.New()
			textArea.CharLimit = 255
			if !foundTextarea {
				textArea.Focus()
				foundTextarea = true
			}
			themodel.textAreas = append(themodel.textAreas, TextArea{textArea, element.Id})
		}
	}
}

func new_element(p *lua.LState) int {
	class := p.ToString(1)
	text := p.ToString(2)
	id := p.ToString(3)
	callable := p.ToString(4)
	placeholder := p.ToString(5)
	p.Push(luar.New(p, Element{class, text, id, callable, placeholder}))
	return 1
}

func add_element(p *lua.LState) int {
	myElement := p.ToUserData(1).Value
	//myelement := myInterfacedElement.(Element)
	thedocument.Elements = append(thedocument.Elements, myElement.(Element))
	return 0
}

func get_by_Id(p *lua.LState) int {
	Id := p.ToString(1)
	for _, element := range thedocument.Elements {
		if element.Id == Id {
			p.Push(luar.New(p, element))
		}
	}
	return 1
}

func set_by_Id(p *lua.LState) int {
	Id := p.ToString(1)
	AAAA := p.ToUserData(2).Value
	newElement := AAAA.(Element)
	for i, element := range thedocument.Elements {
		if element.Id == Id {
			thedocument.Elements[i] = newElement
		}
	}
	return 0
}

func get_document(p *lua.LState) int {
	p.Push(luar.New(p, thedocument))
	return 1
}

func log(text string) {
	client := &http.Client{}
	u, err := url.Parse(os.Getenv("LOGAPP_URI") + "/v1/log")
	if err != nil {
		fmt.Println("Unexpected Error: ", err)
		os.Exit(1)
	}
	q := u.Query()
	q.Add("text", text)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		fmt.Println("Error creating request object for logging:", err)
		os.Exit(1)
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("HTTP Log Err:", err)
		os.Exit(1)
	}
	res.Body.Close()
}

func lua_log(p *lua.LState) int {
	text := p.ToString(-1)
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
  <element class="text" id="myid"></element>
  <element class="text">Write a bit about yourself:</element>
  <element class="textarea" id="myarea" callable="area_handler"></element>
</elements>
<script>
add_element(new_element("text", "Hello, from Lua!"))

function area_handler()
  --add_element(new_element("text", "Hi, ".. get_by_Id("myarea").Text))
end

function name_handler()
  local text = "Hello, ".. get_by_Id("name").Text.. "!"
  set_by_Id("myid", new_element("text", text, "myid"))
end
  </script>
  </document>
`
	err := xml.Unmarshal([]byte(XML), &thedocument)
	if err != nil {
		println(err)
		os.Exit(1)
	}
	state.OpenLibs()
	defer state.Close()
	state.Register("log", lua_log)
	state.Register("get_document", get_document)
	state.Register("add_element", add_element)
	state.Register("new_element", new_element)
	state.Register("get_by_Id", get_by_Id)
	state.Register("set_by_Id", set_by_Id)
	err = state.DoString(thedocument.Script)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	createInitialSpecialElements()
	program := tea.NewProgram(themodel)
	program.Run()
}
