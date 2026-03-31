#include <arpa/inet.h>
#include <netinet/in.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>

int main(void) {
  const char *port_env = getenv("PORT");
  int port = port_env ? atoi(port_env) : 8080;
  int server_fd = socket(AF_INET, SOCK_STREAM, 0);
  int opt = 1;
  setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

  struct sockaddr_in addr;
  memset(&addr, 0, sizeof(addr));
  addr.sin_family = AF_INET;
  addr.sin_addr.s_addr = INADDR_ANY;
  addr.sin_port = htons((unsigned short)port);

  bind(server_fd, (struct sockaddr *)&addr, sizeof(addr));
  listen(server_fd, 16);

  const char *body = "c-basic smoke app";
  char response[256];
  snprintf(
      response,
      sizeof(response),
      "HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %zu\r\nConnection: close\r\n\r\n%s",
      strlen(body),
      body);

  while (1) {
    int client_fd = accept(server_fd, NULL, NULL);
    if (client_fd < 0) {
      continue;
    }
    char buffer[1024];
    (void)read(client_fd, buffer, sizeof(buffer));
    (void)write(client_fd, response, strlen(response));
    close(client_fd);
  }

  return 0;
}
