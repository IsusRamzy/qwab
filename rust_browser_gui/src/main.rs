use iced::{
    Element,
    widget::{button, column, text},
};

fn main() -> iced::Result {
    iced::application("My First App", MyApp::update, MyApp::view).run()
}

struct MyApp {
    state: bool,
}

impl Default for MyApp {
    fn default() -> Self {
        MyApp::new()
    }
}

#[derive(Debug, Clone, PartialEq)]
enum Message {
    Press,
}

impl MyApp {
    fn new() -> Self {
        Self { state: false }
    }

    fn update(&mut self, message: Message) {
        if message == Message::Press {
            self.state = !self.state
        }
    }

    fn view(&self) -> Element<Message> {
        column!(
            text("Hello World!".to_string()),
            button(text("Click ME")).on_press(Message::Press),
            text(format!("{:?}", self.state))
        )
        .into()
    }
}
