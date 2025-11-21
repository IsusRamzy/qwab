# Specification
## How To Transfer the Document
Servers: Return an HTTP response with a text body containing the XML document.
Client: Renders an XML document based on the server's response, then run the script.
## XML
The XML document's base is this:
```xml
<title>TITLE (optional)</title>
<elements>
<element class="text, input, textarea, .." id="exampleID" callable="luaGlobalFunctionToCall"  placeholder="for input and textarea">innerXML for Text</element>
...
</elements>
<script>
-- Lua code
</script>
```
NOTE: An `id` should be used to identify the element for rendering (`input`s, `textarea`s).

### Element Types
- **`text`**: A text element, like the `<p>` tag in HTML.
- **`input`**: An input field, like the `<input>` tag in HTML. Needs `id`. Can have `placeholder`.
- **`textarea`**: Like input but more lines.
- **`button`**: A button that when pressed, calls `callable`. Recommended to have an `id`.
## Lua Scripting
Scripting can be done with Lua using the `<script>` tag.
To modify the document, these global functions are available:
- **`get_by_Id(Id)`**: A function that takes in an ID and returns the element. You can access: `Id`, `Class`, `Callable`, and `Placeholder`, `Text` where `Text` represents the user input in case of an input/textarea.
- **`set_by_Id(Id, element)`**: A function that takes in an ID and an element (one can create an element using `new_element()`), then sets it. NOTE: Provide an ID to `new_element()`.
- **`new_element(class, text, id, callable, placeholder)`**: Returns a new element based on the provided parameters. Parameters that aren't provided shall be "zeroed".
- **`add_element(element)`**: Adds an element to the document.
- **`set_cookie(key, val)`**: Sets cookie with the key `key` to `val`. NOTE: cookies are stored by host name.
- **`get_cookie(key)`**: Gets cookie with the key `key`.
- **`log(mystring)`**: Logs `mystring` to a console.
- **``POST(server_url, mystring)``**: Sends a `POST` request to `server_url` with the request's body as `mystring` and returns the response's body.
- **`GET(server_url)`**: Sends a `GET` request to `server_url` and returns the response's body.
And these are the functions of the library `json`:
- **`json.encode(mydata)`** Encodes `mydata` into JSON. 
- **`json.decode(myjson)`** Decodes `myjson` from JSON into data.

