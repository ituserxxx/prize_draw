package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	qrcode "github.com/skip2/go-qrcode"
	"log"
	"net/http"
	"os"
	"sync"
)

var cache sync.Map

/*
	设置缓存

cache.Store("key", "value")
// 获取缓存
value, found := cache.Load("key")

	if found {
		fmt.Println(value)
	}

// 删除缓存
cache.Delete("key")
*/
func main() {

	fmt.Println("http://ip:8001")
	// 启动HTTP服务器
	http.ListenAndServe(":8001", RegisterRoute())
}

func RegisterRoute() http.Handler {
	router := http.NewServeMux()
	handler := corsMiddleware(router)
	fs := http.FileServer(http.Dir("dist"))
	//建立路由规则，将所有请求交给静态文件处理器处理
	router.Handle("/cj_qrcode", fs)
	// GET请求处理

	// ws 请求
	router.HandleFunc("/ws", middleware(wsHandler))

	// POST请求处理
	router.HandleFunc("/hello", middleware(apiHandlerTest))                                // 测试
	router.HandleFunc("/api/get_cj_info", middleware(Crontroller.HandlerGetCjInfo))        // 获取抽奖信息
	router.HandleFunc("/api/save_cj_info", middleware(Crontroller.HandlerGetCjInfo))       // 保存抽奖信息
	router.HandleFunc("/api/save_gs_name", middleware(Crontroller.HandlerGetCjInfo))       // 保存公司名称
	router.HandleFunc("/api/get_join_cj_number", middleware(Crontroller.HandlerGetCjInfo)) // 参与抽奖
	router.HandleFunc("/api/send_cj_dyamic_msg", middleware(Crontroller.HandlerGetCjInfo)) // 发送抽奖互动消息
	return handler
}

var Crontroller = &uLogic{}

type uLogic struct {
}

func (uc *uLogic) HandlerGetCjInfo(w http.ResponseWriter, r *http.Request) {

}

type cjOption struct {
	Name string `json:"name"` // 奖品名称	Level int    `json:"level"` // 几等奖
	Num  int    `json:"num"`  //奖品数量
}
type cj struct {
	CJId        string     `json:"cj_id"` // uuid当做cjid
	GsName      string     `json:"gs_name"`
	CjQrcode    string     `json:"cj_qrcode"`
	PersonTotal int        `json:"person_total"` //当前参与人数
	L           []cjOption `json:"l"`
}

var cjManager = map[string]cj{}

// 自定义中间件函数
func middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("uuid") == "" {
			http.Error(w, " Not Allowed Connect ", http.StatusBadRequest)
			return
		}
		// 调用下一个处理函数
		next(w, r)
	}
}
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置允许跨域的域名列表
		// 设置允许的请求头部
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,X-token")

		// 对于预检请求（OPTIONS），直接返回成功
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 继续执行下一个处理程序
		next.ServeHTTP(w, r)
	})
}
func apiHandlerTest(w http.ResponseWriter, r *http.Request) {
	// 获取路由参数
	name := r.URL.Query().Get("name")
	respOk(w, name)
	respErr(w, name)
}

type resp struct {
	Data interface{} `json:"data"`
	Code int         `json:"code"`
}

func respHandle(w http.ResponseWriter, code int, data interface{}) ([]byte, error) {
	response := resp{
		Data: data,
		Code: code,
	}
	w.Header().Set("Content-Type", "application/json") // 设置响应头的Content-Type为"application/json"
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	return jsonResponse, nil
}
func respOk(w http.ResponseWriter, data interface{}) {
	jsonResponse, err := respHandle(w, 33, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(jsonResponse))

}
func respErr(w http.ResponseWriter, data string) {
	jsonResponse, err := respHandle(w, 44, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(w, string(jsonResponse))
}

/*---------qrcode--------------*/
func createQrcodeUrl(uuid string) (string, error) {
	fileName := fmt.Sprintf("cj_qrcode/%s.png", uuid)

	// 生成二维码
	err := qrcode.WriteFile(fileName, qrcode.Medium, 256, fileName)
	if err != nil {
		return "", err
	}
	return "http://www.cj.yirisanqiu.com/" + fileName, nil
}
func getQrcodeUrl(uuid string) string {
	k := "cj_qrcode-" + uuid
	value, found := cache.Load(k)
	if found {
		return fmt.Sprintf("%v", value)
	}
	url, err := createQrcodeUrl(uuid)
	if err != nil {
		fmt.Println("createQrcodeUrl", err.Error())
		return ""
	}
	cache.Store(k, url)
	return url
}
func delQrcodeUrl(uuid string) {
	k := "cj_qrcode" + uuid
	cache.Delete(k)
	os.Remove(fmt.Sprintf("cj_qrcode/%s.png", uuid))
}

/* ---------ws------------*/
type ClientManager struct {
	Clients    map[string]*WsClient
	Broadcast  chan []byte
	Register   chan *WsClient
	Unregister chan *WsClient
}

type WsClient struct {
	Socket *websocket.Conn
	Send   chan []byte
	UUID   string
}

var Manager = ClientManager{
	Register:   make(chan *WsClient),
	Unregister: make(chan *WsClient),
	Clients:    make(map[string]*WsClient),
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade:", err)
		return
	}
	defer conn.Close()
	uuid := r.Header.Get("uuid")
	//可以添加用户信息验证
	client := &WsClient{
		Socket: conn,
		Send:   make(chan []byte),
		UUID:   uuid,
	}
	// 当前抽奖不存在
	if _, ok := cjManager[uuid]; ok == false {
		cjManager[uuid] = cj{
			CJId:        uuid,
			GsName:      "",
			CjQrcode:    getQrcodeUrl(uuid),
			PersonTotal: 0,
			L:           nil,
		}
	}
	Manager.Register <- client
	go client.Read()
	go client.Write()
}
func (c *WsClient) Read() {
	defer func() {
		Manager.Unregister <- c
		_ = c.Socket.Close()
	}()
	for {
		c.Socket.PongHandler()
		_, message, err := c.Socket.ReadMessage()
		if err != nil {
			_ = c.Socket.Close()
			break
		}
		fmt.Println(string(message))
		if len(message) == 0 {
			c.Send <- message
			continue
		}
		fmt.Println(string(message))
	}
}
func (c *WsClient) Write() {
	defer func() {
		Manager.Unregister <- c
		_ = c.Socket.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				_ = c.Socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			_ = c.Socket.WriteMessage(websocket.TextMessage, message)
		}
	}
}

// WSStart is  项目运行前, 协程开启start -> go Manager.Start()
func (manager *ClientManager) WSStart() {
	for {
		select {
		case conn := <-Manager.Register:
			Manager.Clients[conn.UUID] = conn
		case conn := <-Manager.Unregister:
			if _, ok := Manager.Clients[conn.UUID]; ok {
				close(conn.Send)
				delete(Manager.Clients, conn.UUID)
			}
		}
	}
}
func SendToWeb(uuid string, data interface{}) {
	for _, c := range Manager.Clients {
		if c.UUID == uuid {
			b, _ := json.Marshal(data)
			c.Send <- b
		}
	}
}
