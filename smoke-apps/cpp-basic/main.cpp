#include <arpa/inet.h>
#include <cstdlib>
#include <cstring>
#include <netinet/in.h>
#include <string>
#include <sys/socket.h>
#include <unistd.h>

int main() {
  const char* port_env = std::getenv("PORT");
  int port = port_env ? std::atoi(port_env) : 8080;

  int server_fd = socket(AF_INET, SOCK_STREAM, 0);
  int opt = 1;
  setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

  sockaddr_in addr{};
  addr.sin_family = AF_INET;
  addr.sin_addr.s_addr = INADDR_ANY;
  addr.sin_port = htons(static_cast<unsigned short>(port));

  bind(server_fd, reinterpret_cast<sockaddr*>(&addr), sizeof(addr));
  listen(server_fd, 16);

  const std::string body = "cpp-basic smoke app";
  const std::string response =
      "HTTP/1.1 200 OK\r\n"
      "Content-Type: text/plain; charset=utf-8\r\n"
      "Content-Length: " +
      std::to_string(body.size()) +
      "\r\nConnection: close\r\n\r\n" + body;

  while (true) {
    int client_fd = accept(server_fd, nullptr, nullptr);
    if (client_fd < 0) {
      continue;
    }
    char buffer[1024];
    (void)read(client_fd, buffer, sizeof(buffer));
    (void)write(client_fd, response.c_str(), response.size());
    close(client_fd);
  }

  return 0;
}
