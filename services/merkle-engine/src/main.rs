use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::thread;
use std::time::{Duration, Instant};

fn handle_client(mut stream: TcpStream, started: Instant) {
    let mut buf = [0u8; 1024];
    let _ = stream.read(&mut buf);
    let request = String::from_utf8_lossy(&buf);
    let first_line = request.lines().next().unwrap_or_default();

    let (status_line, content_type, body) = if first_line.starts_with("GET /healthz") {
        ("HTTP/1.1 200 OK\r\n", "text/plain", "ok\n".to_string())
    } else if first_line.starts_with("GET /metrics") {
        let uptime = started.elapsed().as_secs_f64();
        let metrics = format!(
            "# HELP merkle_engine_uptime_seconds Process uptime in seconds.\n# TYPE merkle_engine_uptime_seconds gauge\nmerkle_engine_uptime_seconds {}\n# HELP merkle_engine_tree_size Current tree size (placeholder).\n# TYPE merkle_engine_tree_size gauge\nmerkle_engine_tree_size 0\n",
            uptime
        );
        ("HTTP/1.1 200 OK\r\n", "text/plain; version=0.0.4", metrics)
    } else {
        (
            "HTTP/1.1 404 Not Found\r\n",
            "text/plain",
            "not found\n".to_string(),
        )
    };

    let response = format!(
        "{}Content-Type: {}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
        status_line,
        content_type,
        body.len(),
        body
    );
    let _ = stream.write_all(response.as_bytes());
}

fn main() {
    let addr = std::env::var("METRICS_ADDR").unwrap_or_else(|_| "0.0.0.0:8000".to_string());
    let started = Instant::now();

    let listener = TcpListener::bind(&addr).expect("failed to bind metrics listener");
    println!("merkle-engine starting metrics/health endpoint on {}", addr);

    for stream in listener.incoming() {
        match stream {
            Ok(stream) => handle_client(stream, started),
            Err(err) => eprintln!("accept error: {}", err),
        }
        thread::sleep(Duration::from_millis(5));
    }
}
