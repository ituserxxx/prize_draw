package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {

	fmt.Println("http://ip:8001")
	// 启动HTTP服务器
	http.ListenAndServe(":8001", RegisterRoute())

}

func RegisterRoute() http.Handler {
	router := http.NewServeMux()
	handler := corsMiddleware(router)
	// fs := http.FileServer(http.Dir("dist"))
	// 建立路由规则，将所有请求交给静态文件处理器处理
	// router.Handle("/", fs)
	// GET请求处理
	router.HandleFunc("/hello", helloHandler)
	// POST请求处理
	router.HandleFunc("/user/Login", middleware(helloHandler))
	router.HandleFunc("/user/Info", middleware(helloHandler))
	return handler
}

// 自定义中间件函数
func middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
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
func helloHandler(w http.ResponseWriter, r *http.Request) {
	// 获取路由参数
	name := r.URL.Query().Get("name")

	// 返回响应
	fmt.Fprintf(w, "Hello, %s!", name)
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

var Crontroller = &uLogic{}

type uLogic struct {
}

type getCjInfoReq struct {
	CjId int `json:"cj_id"`
}

func (uc *uLogic) getCjInfo(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		respErr(w, err.Error())
		return
	}

	var params getCjInfoReq
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respErr(w, err.Error())
		return
	}
	respOk()
}
