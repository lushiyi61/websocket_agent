package main

import (
    "io"
    "log"
    "net"
    // "os"
    // "fmt"
    "bufio"
    "bytes"
    "strconv"
    "strings"
    "crypto/sha1"
    "encoding/base64"
    "encoding/binary"
    "runtime"
    "net/http"
)

type SocketType byte
const (
    EWebSocket SocketType = iota   // value --> 0
    ETcpSocket                     // value --> 1
)

type TcpSocket struct {
    Listener    net.Listener
    Clients        []*Client
    socketType     SocketType   
}

type Client struct {
    Conn         net.Conn
    Nickname    string
    Shook        bool
    Server        *TcpSocket
    Id            int
    TcpConn      net.Conn
    WebsocketType   int
    Num             int
    socketType     SocketType   
}

type Msg struct {
    Data        string
    Num            int
}

func (self *Client) Release() {
    // release all connect
    self.TcpConn.Close()
    self.Conn.Close()
}

// 连接初始化 需要支持websocket&socket
func (self *Client) Handle() {
    log.Println("Handle,socketType：", self.socketType)
    defer self.Release()

    if(self.socketType == EWebSocket){
    if !self.Handshake() {
            log.Fatalln("Handshake err , del this conn")
        return
    }
    }

    // connect to another server for tcp
    if !self.ConnTcpServer(){
        log.Fatalln("can not connect to the other server , release")
        return
    }
    num = num + 1
    log.Println("now connect num : ", num)
    self.Num = num
    go self.Read()
    self.ReadTcp()
}

func (self *Client) ReadFromTcpSocket() {
    request := make([]byte, 128)
    for {
		read_len, err := self.Conn.Read(request)

		if err != nil {
			log.Println(err)
			break
		}

		if read_len == 0 {
			break // connection already closed by client
		}  else {
			buf := request[:read_len]
            log.Println("rec from the client(",self.Num,")", string(buf))
            self.TcpConn.Write(buf)
		}
        request = make([]byte, 128) // clear last read content
	}
}

func (self *Client) ReadFromWebSocket() {
    var (
        buf     []byte
        mKey    []byte
        length    uint64
        l        uint16
    )
    for {
        buf = make([]byte, 2)
        _, err := io.ReadFull(self.Conn, buf)
        if err != nil {
            log.Println(err)
            self.Release()
            break
        }
        //fin = buf[0] >> 7
        //if fin == 0 {
        //}
        rsv := (buf[0] >>4) &0x7
        // which must be 0
        if rsv != 0{
            log.Println("Client send err msg:",rsv,", disconnect it")
            self.Release()
            break
        }

        opcode := buf[0] & 0xf
        // opcode   if 8 then disconnect
        if opcode == 8 {
            log.Println("CLient want close Connection")
            self.Release()
            break
        }

        // should save the opcode 
        // if client send by binary should return binary (especially for Egret)
        self.WebsocketType = int(opcode)

        mask := buf[1] >> 7
        // the translate may have mask 

        payload := buf[1] & 0x7f
        // if length < 126 then payload mean the length
        // if length == 126 then the next 8bit mean the length
        // if length == 127 then the next 64bit mean the length
        switch {
            case payload < 126:
                length = uint64(payload)

            case payload == 126:
                buf = make([]byte, 2)
                io.ReadFull(self.Conn, buf)
                binary.Read(bytes.NewReader(buf), binary.BigEndian, &l)
                length = uint64(l)

            case payload == 127:
                buf = make([]byte, 8)
                io.ReadFull(self.Conn, buf)
                binary.Read(bytes.NewReader(buf), binary.BigEndian, &length)
        }
        if mask == 1 {
            mKey = make([]byte, 4)
            io.ReadFull(self.Conn, mKey)
        }
        buf = make([]byte, length)
        io.ReadFull(self.Conn, buf)
        if mask == 1 {
            for i, v := range buf {
                buf[i] = v ^ mKey[i % 4]
            }
            //fmt.Println("mask", mKey)
        }
              
        log.Println("rec from the client(",self.Num,")", string(buf))
        self.TcpConn.Write(buf)
    }
}

// Read 需要支持websocket&socket
func (self *Client) Read() {
    if(self.socketType == EWebSocket){
        self.ReadFromWebSocket();
    } else {
        self.ReadFromTcpSocket();
    } 
}

// read from server tcp
func (self *Client) ReadTcp() {
    var (
        buf  []byte
    )
    buf = make([]byte, 1024)

    for {
        length,err := self.TcpConn.Read(buf)

        if err != nil {
            self.Release()
            num = num - 1
            // only 
            log.Println("other tcp connect err", err)
            log.Println("disconnect client :", self.Num)
            log.Println("now have:", num)
            break
        }
        log.Println("recv from other tcp : ", string(buf[:length]))
        self.Write(buf[:length])
    }
}

// write 需要支持websocket&socket
func (self *Client) Write(data []byte) bool {
    if(self.socketType == EWebSocket){
    data_binary := new(bytes.Buffer) //which 

    //should be binary or string
    frame := []byte{129}  //string
    length := len(data)
    // 10000001
    if self.WebsocketType == 2 {
        frame = []byte{130}
        // 10000010
        err := binary.Write(data_binary, binary.LittleEndian, data)
        if err != nil {
                    log.Println(" translate to binary err", err)
        }
        length = len(data_binary.Bytes())
    }
    switch {
    case length < 126:
        frame = append(frame, byte(length))
    case length <= 0xffff:
        buf := make([]byte, 2)
        binary.BigEndian.PutUint16(buf, uint16(length))
        frame = append(frame, byte(126))
        frame = append(frame, buf...)
    case uint64(length) <= 0xffffffffffffffff:
        buf := make([]byte, 8)
        binary.BigEndian.PutUint64(buf, uint64(length))
        frame = append(frame, byte(127))
        frame = append(frame, buf...)
    default:
                log.Println("Data too large")
        return false
    }
    if self.WebsocketType == 2 {
        frame = append(frame, data_binary.Bytes()...)
    } else {
        frame = append(frame, data...)
    }
    self.Conn.Write(frame)
    } else {
        self.Conn.Write([]byte(data)) 
    }

    // frame = []byte{0}
    return true
}

// 连接到TCP服务器
func (self *Client) ConnTcpServer() bool {

    conn, err := net.Dial("tcp", tcpServerAddr)

    if(err != nil) {
        log.Println("connect other tcp server error")
        return false
    }

    self.TcpConn = conn
    return true
}

// 处理websocket 握手问题
func (self *Client) Handshake() bool {
    if self.Shook {
        return true
    }
    reader := bufio.NewReader(self.Conn)
    key := ""
    str := ""
    for {
        line, _, err := reader.ReadLine()
        if err != nil {
            log.Println("Handshake err:", err)
            return false
        }
        if len(line) == 0 {
            break
        }
        str = string(line)
        if strings.HasPrefix(str, "Sec-WebSocket-Key") {
            if len(line)>= 43 {
                key = str[19:43]
            }
        }
    }
    if key == "" {
        return false
    }
    sha := sha1.New()
    io.WriteString(sha, key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
    key = base64.StdEncoding.EncodeToString(sha.Sum(nil))
    header := "HTTP/1.1 101 Switching Protocols\r\n" +
    "Connection: Upgrade\r\n" +
    "Sec-WebSocket-Version: 13\r\n" +
    "Sec-WebSocket-Accept: " + key + "\r\n" +
    "Upgrade: websocket\r\n\r\n"
    self.Conn.Write([]byte(header))
    self.Shook = true
    self.Server.Clients = append(self.Server.Clients, self)
    return true
}

// 新建立连接
func NewSocket(addr string, sType SocketType) *TcpSocket {
    log.Println("NewSocket:", addr)
    listener, err := net.Listen("tcp", addr)
    if err != nil {
        log.Fatal(err)
    }
    return &TcpSocket{listener, make([]*Client, 0), sType}
}

// 监听客户端消息
func (self *TcpSocket) Loop() {
    for {
        conn, err := self.Listener.Accept()
        if err != nil {
            log.Println("client conn err:", err)
            continue
        }
        s := conn.RemoteAddr().String()
        i, _ := strconv.Atoi(strings.Split(s, ":")[1])
        log.Println("new Client:", s)

        client := &Client{conn, "", false, self, i, conn, 1, num, self.socketType}
        go client.Handle()
    }
}


func handler(w http.ResponseWriter, r *http.Request) {
    // show num of goroutine
    w.Header().Set("Content-Type", "text/plain")
    num := strconv.FormatInt(int64(runtime.NumGoroutine()), 10)
    w.Write([]byte(num))
}


// ============主函数============
var webSocketPort = "11305"
var tcpSocketPort = "11306"
var IPAddress = "0.0.0.0:"
var num = 0
var tcpServerAddr = "127.0.0.1:12306"
func main() {

    // webSocket
    ws := NewSocket(IPAddress + string(webSocketPort), EWebSocket)
    go  ws.Loop()

    // tcpSocket
    ts := NewSocket(IPAddress + string(tcpSocketPort), ETcpSocket)
    go  ts.Loop()

    log.Println("====Start Listen====")
    // // listen 11181 to show num of goroutine
    http.HandleFunc("/", handler)
    http.ListenAndServe(":11181", nil)
}

