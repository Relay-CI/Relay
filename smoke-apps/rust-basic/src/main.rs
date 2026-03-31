use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};

fn main() {
    let port = std::env::var("PORT").unwrap_or_else(|_| "8080".to_string());
    let listener = TcpListener::bind(format!("0.0.0.0:{port}")).expect("bind failed");

    for stream in listener.incoming() {
        match stream {
            Ok(stream) => handle(stream),
            Err(err) => eprintln!("accept error: {err}"),
        }
    }
}

fn handle(mut stream: TcpStream) {
    let mut buffer = [0_u8; 1024];
    let _ = stream.read(&mut buffer);

    let body = b"rust-basic smoke app";
    let response = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        body.len()
    );

    let _ = stream.write_all(response.as_bytes());
    let _ = stream.write_all(body);
    let _ = stream.flush();
}
