package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
	luar "layeh.com/gopher-luar"
)

var httpclient *http.Client = &http.Client{}
var state *lua.LState = lua.NewState()
var thedocument Document = Document{}
var themodel model = model{}
var thecookies map[string]string

const PERMS = 0775

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
	if themodel.cursor == findIndexOfElementById(Id) {
		return true
	}
	return false
}

func callSpecialCallable(callable string) {
	whatever := state.GetGlobal(callable)
	state.Push(whatever)
	state.CallByParam(lua.P{
		Fn:   state.GetGlobal(callable),
		NRet: 0,
	}, nil)

}

func callAllSpecialCallables() {
	for _, element := range thedocument.Elements {
		if (element.Class == "textarea" || element.Class == "input") && isFocused(element.Id) {
			if element.Callable != "" {
				callSpecialCallable(element.Callable)
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
		} else if msg.Type == tea.KeyEnter {
			for i, elem := range thedocument.Elements {
				if i == themodel.cursor && elem.Class == "button" {
					callSpecialCallable(elem.Callable)
				}
			}
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
		} else if element.Class == "button" {
			text += "[" + element.Text + "]"
			text += "\n"
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

func getXmlCode() string {
	u, err := url.Parse(os.Args[1])
	if err != nil {
		fmt.Println("Unexpected Error: ", err)
		os.Exit(1)
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		fmt.Println("Error creating request object:", err)
		os.Exit(1)
	}
	res, err := httpclient.Do(req)
	if err != nil {
		fmt.Println("HTTP Err:", err)
		os.Exit(1)
	}
	mybody, _ := io.ReadAll(res.Body)
	res.Body.Close()
	return "<document>" + string(mybody) + "</document>"
}

func log(text string) {
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
	res, err := httpclient.Do(req)
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

func getCookiesPath() string {
	home, _ := os.UserHomeDir()
	parsed, _ := url.Parse(os.Args[1])
	os.MkdirAll(home+"/.local/usr/share/qwab/cookies/", PERMS)
	return home + "/.local/usr/share/qwab/cookies/" + parsed.Host + ".json"
}

func loadCookies() {
	FILEPATH := getCookiesPath()
	_, err := os.Stat(FILEPATH)
	var fileExists bool
	if err != nil {
		fileExists = false
	} else if err == nil {
		fileExists = true
	}
	if fileExists {
		data, _ := os.ReadFile(FILEPATH)
		json.Unmarshal(data, &thecookies)
	} else {
		thecookies = map[string]string{}
		os.WriteFile(FILEPATH, []byte(`{}`), PERMS)
	}
}

func updateCookies() {
	cookiesPath := getCookiesPath()
	marshaled, _ := json.Marshal(thecookies)
	os.WriteFile(cookiesPath, marshaled, PERMS)
}

func set_cookie(p *lua.LState) int {
	key := p.ToString(1)
	val := p.ToString(2)
	thecookies[key] = val
	updateCookies()
	return 0
}

func get_cookie(p *lua.LState) int {
	key := p.ToString(1)
	p.Push(lua.LString(thecookies[key]))
	return 1
}

func GET(p *lua.LState) int {
	http_server := p.ToString(1)
	req, err := http.NewRequest("GET", http_server, nil)
	if err != nil {
		p.Push(lua.LString("ERR_CREATING_REQUEST"))
		return 1
	}
	res, err := httpclient.Do(req)
	if err != nil {
		p.Push(lua.LString("ERR_SENDING_REQUEST"))
		return 1
	}
	mybody, _ := io.ReadAll(res.Body)
	res.Body.Close()
	p.Push(lua.LString(mybody))
	return 1
}

func main() {
	XML := getXmlCode()
	err := xml.Unmarshal([]byte(XML), &thedocument)
	if err != nil {
		fmt.Println("XML error:", err)
		os.Exit(1)
	}
	loadCookies()
	state.OpenLibs()
	defer state.Close()
	luajson.Preload(state)
	state.Register("log", lua_log)
	state.Register("add_element", add_element)
	state.Register("new_element", new_element)
	state.Register("get_by_Id", get_by_Id)
	state.Register("set_by_Id", set_by_Id)
	state.Register("get_cookie", get_cookie)
	state.Register("set_cookie", set_cookie)
	state.Register("GET", GET)
	err = state.DoString(thedocument.Script)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	createInitialSpecialElements()
	program := tea.NewProgram(themodel)
	program.Run()
}
