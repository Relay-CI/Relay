const http = require("http");

const port = Number(process.env.PORT || 3000);

const server = http.createServer((req, res) => {
  res.writeHead(200, { "content-type": "text/plain; charset=utf-8" });
  res.end("node-basic smoke app");
});

server.listen(port, "0.0.0.0", () => {
  console.log(`node-basic listening on ${port}`);
});
