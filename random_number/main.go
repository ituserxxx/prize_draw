package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	qrcode "github.com/skip2/go-qrcode"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

func main() {
	fmt.Println("http://ip:8001")
	// 启动HTTP服务器
	http.ListenAndServe(":8001", RegisterRoute())
}

var cache sync.Map

// cache Demo
func cacheDemo() {
	cache.Store("key", "value")
	// 获取缓存
	value, found := cache.Load("key")

	if found {
		fmt.Println(value)
	}
	// 删除缓存
	cache.Delete("key")
}

// qrcode demo
func qrcodeDemo() {
	data := "http://www.cj.yirisanqiu.com/" // 自定义参数的 URL 或文本
	filePath := "index_qrcode.png"          // 存放目录和文件名
	// 生成二维码
	err := qrcode.WriteFile(data, qrcode.Medium, 256, filePath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("二维码生成成功！")
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
	router.HandleFunc("/hello", middleware(apiHandlerTest))                                      // 测试
	router.HandleFunc("/api/get_cj_info", middleware(Crontroller.HandlerGetCjInfo))              // 获取抽奖信息
	router.HandleFunc("/api/save_cj_info", middleware(Crontroller.HandlerSaveCjInfo))            // 保存抽奖信息
	router.HandleFunc("/api/save_gs_name", middleware(Crontroller.HandlerSaveGsName))            // 保存公司名称
	router.HandleFunc("/api/get_join_cj_number", middleware(Crontroller.HandlerJoinCj))          // 参与抽奖
	router.HandleFunc("/api/send_cj_dyamic_msg", middleware(Crontroller.HandlerSendCjDyamicMsg)) // 发送抽奖互动消息
	return handler
}

var Crontroller = &uLogic{}

type uLogic struct {
}

type cjOption struct {
	Name string `json:"name"` // 奖品名称	Level int    `json:"level"` // 几等奖
	Num  int    `json:"num"`  //奖品数量
}
type JoinUser struct {
	UUID     string `json:"uuid"`
	CjNumber int    `json:"cj_number"`
}
type cj struct {
	CJId         string     `json:"cj_id"` // uuid当做cjid
	GsName       string     `json:"gs_name"`
	CjQrcode     string     `json:"cj_qrcode"`
	L            []cjOption `json:"l"`
	JoinUserList []JoinUser // 参与用户列表
	PersonTotal  int        `json:"person_total"` //当前参与人数
	Lock         sync.RWMutex
}

var cjManager = map[string]*cj{}
var cjMLock sync.RWMutex

func (uc *uLogic) HandlerGetCjInfo(w http.ResponseWriter, r *http.Request) {
	var info = &cj{}

	uuid := r.Header.Get("uuid")
	// 当前抽奖不存在
	if v, ok := cjManager[uuid]; ok == false { // 活动发起人的 uuid == lid  抽奖id
		info.CJId = uuid
		info.CjQrcode = getQrcodeUrl(uuid)
		cjMLock.Lock()
		defer cjMLock.Unlock()
		cjManager[uuid] = info
	} else {
		info = v
	}
	respOk(w, info)
}

func (uc *uLogic) HandlerSaveCjInfo(w http.ResponseWriter, r *http.Request) {
	// 读取请求体数据
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respErr(w, "parse req data err "+err.Error())
		return
	}
	// 解析请求体数据到结构体
	var info *cj
	err = json.Unmarshal(body, &info)
	if err != nil {
		respErr(w, "json unmarshal err "+err.Error())
		return
	}
	uuid := r.Header.Get("uuid")
	if v, ok := cjManager[uuid]; ok == false && info != nil {
		cjManager[info.CJId] = info
	} else {
		v.Lock.RLock()
		defer v.Lock.RUnlock()
		v.GsName = info.GsName
		v.L = info.L
	}
	respOk(w, "")
}
func (uc *uLogic) HandlerSaveGsName(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		respErr(w, "parse req data err "+err.Error())
		return
	}

	// 获取 POST 参数
	name := strings.TrimSpace(r.Form.Get("gs_name"))
	if len(name) == 0 {
		respErr(w, "活动名称不能为空")
		return
	}
	uuid := r.Header.Get("uuid")
	if v, ok := cjManager[uuid]; ok == false {
		respErr(w, "cj 不存在 "+err.Error())
		return
	} else {
		v.Lock.RLock()
		defer v.Lock.RUnlock()
		v.GsName = name
	}
	respOk(w, "")
}
func (uc *uLogic) HandlerJoinCj(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		respErr(w, "parse req data err "+err.Error())
		return
	}
	// 获取 POST 参数
	lid := strings.TrimSpace(r.Form.Get("lid"))
	if len(lid) == 0 {
		respErr(w, "活动id不能为空")
		return
	}
	uuid := r.Header.Get("uuid")
	if v, ok := cjManager[lid]; ok == false {
		respErr(w, "cj 不存在 "+err.Error())
		return
	} else {
		v.Lock.RLock()
		defer v.Lock.RUnlock()
		v.PersonTotal += 1
		for _, u := range v.JoinUserList {
			if u.UUID == uuid {
				respOk(w, "")
				return
			}
		}
		v.JoinUserList = append(v.JoinUserList, JoinUser{
			UUID:     uuid,
			CjNumber: v.PersonTotal,
		})
	}
	respOk(w, "")
}
func (uc *uLogic) HandlerSendCjDyamicMsg(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		respErr(w, "parse req data err "+err.Error())
		return
	}
	// 获取 POST 参数
	lid := strings.TrimSpace(r.Form.Get("lid"))
	if len(lid) == 0 {
		respErr(w, "活动id不能为空")
		return
	}
	msg := strings.TrimSpace(r.Form.Get("msg"))
	if len(msg) == 0 {
		respErr(w, "")
		return
	}
	uuid := r.Header.Get("uuid")
	// 获取当当前用户的编号
	var uNumber int
	v, ok := cjManager[lid]
	if ok == false {
		respErr(w, "cj 不存在 "+err.Error())
		return
	}
	for _, u := range v.JoinUserList {
		if u.UUID == uuid {
			uNumber = u.CjNumber
			break
		}
	}
	if uNumber == 0 {
		respErr(w, "请先参与该活动")
		return
	}
	_, isTrue := Manager.Clients[lid]
	if isTrue {
		SendToWeb(lid, msg)
	}

}

// 自定义中间件函数
func middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.TrimSpace(r.Header.Get("uuid")) == "" {
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,X-token,uuid")

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
