package main
import (
	"fmt"
	"io"
	"time"
	//"io/ioutil"
	"net/url"
	"net/http"
	"encoding/json"
	"github.com/zaddone/studySystem/request"
	"github.com/lunny/html2md"
	"github.com/gin-gonic/gin"
	"flag"
)
var (
	Router  = gin.Default()
	searchzhihuUrl *url.URL
	port = flag.String("p","8080","port")
)

func init(){
	flag.Parse()
	var err error
	searchzhihuUrl,err = url.Parse("https://api.zhihu.com/search_v3?advert_count=0&correction=1&lc_idx=0&limit=20&offset=20&q=%E5%9B%B4%E6%A3%8B&show_all_topics=0&t=general")
	if err != nil {
		panic(err)
	}
	Router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK,gin.H{"msg":"success"})
	})
	//Router.POST("/update", func(c *gin.Context) {
	//	db, err := ioutil.ReadAll(c.Request.Body)
	//	if err != nil {
	//		panic(err)
	//	}
	//})
	Router.GET("/up", func(c *gin.Context) {
		err = run()
		if err != nil {
			//fmt.Println(err)
			//c.JSON(http.StatusFound,err)
			c.JSON(http.StatusNotFound,err)
			return
		}
		c.JSON(http.StatusOK,gin.H{"msg":"success"})
	})
	Router.GET("/search", func(c *gin.Context) {
		var li []interface{}
		err := searchWithWords(c.Query("q"),20,func(o interface{}){
			li = append(li,o)
		})
		if err != nil {
			fmt.Println(err)
			c.JSON(http.StatusNotFound,err)
			return
		}
		c.JSON(http.StatusOK,li)
	})
	go Router.Run(":"+*port)
	//return
	go func () {
		for{

			err = run()
			if err != nil {
				fmt.Println(err)
			}

			err = runRead()
			if err != nil {
				fmt.Println(err)
			}

			time.Sleep(time.Hour*1)
		}
	}()

}
func initZhihu()error{
	return request.ClientHttp__("https://www.zhihu.com/udid","POST",nil,nil,func(body io.Reader,res *http.Response)error{
		if res.StatusCode != 200 {
			return fmt.Errorf(res.Status)
		}
		//db,err := ioutil.ReadAll(body)
		//if err != nil {
		//	return err
		//}
		//fmt.Println(db)
		//h(string(db))
		return nil

	})
}
func searchZhihu(word string,h func(interface{}))error{
	query := searchzhihuUrl.Query()
	query.Set("q",word)
	//u := fmt.Sprintf("%s://%s%s?%s",searchzhihuUrl.Scheme,searchzhihuUrl.Host,searchzhihuUrl.Path,query.Encode())
	//fmt.Println(u)
	return request.ClientHttp__(fmt.Sprintf("%s://%s%s?%s",searchzhihuUrl.Scheme,searchzhihuUrl.Host,searchzhihuUrl.Path,query.Encode()),"GET",nil,nil,func(body io.Reader,res *http.Response)error{
		if res.StatusCode != 200 {
			return fmt.Errorf(res.Status)
		}
		//db,err := ioutil.ReadAll(body)
		var db interface{}
		err := json.NewDecoder(body).Decode(&db)
		if err != nil {
			return err
		}
		for _,d := range db.(map[string]interface{})["data"].([]interface{}) {
			obj := d.(map[string]interface{})["object"].(map[string]interface{})
			que := obj["question"]
			if que == nil {
				continue
			}
			title := que.(map[string]interface{})["name"]
			if title == nil {
				continue
			}
			cou := obj["content"]
			if cou == nil {
				continue
			}
			author := obj["author"]
			if author == nil {
				continue
			}
			c,err := NewContentZhihu(html2md.Convert(title.(string)),html2md.Convert(cou.(string)),author.(map[string]interface{})["name"].(string))
			if err != nil {
				fmt.Println(err)
			}
			err = c.saveWithDB(false,nil)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = c.addSame()
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(c.showId(),c.words)
			err = c.saveWordsWithDB()
			if err != nil {
				fmt.Println(err)
			}
			h(c)
		}
		//h(db)
		return nil

	})
}
func hotZhihu(h func(interface{}))error{
	return request.ClientHttp__("https://www.zhihu.com/api/v4/search/top_search","GET",nil,nil,func(body io.Reader,res *http.Response)error{
		if res.StatusCode != 200 {
			return fmt.Errorf(res.Status)
		}
		var db interface{}
		err := json.NewDecoder(body).Decode(&db)
		if err != nil {
			return err
		}
		search := db.(map[string]interface{})["top_search"]
		if search == nil {
			return fmt.Errorf("top_search is nil")
		}
		fmt.Println(db)
		for _,w := range search.(map[string]interface{})["words"].([]interface{}){
			h(w.(map[string]interface{})["display_query"])
		}
		return nil
	})
}

func run()error{
	//fmt.Println("ok")
	err := initZhihu()
	if err != nil {
		return err
	}
	return hotZhihu(func(words interface{}){
		fmt.Println(words)
		err :=  searchZhihu(words.(string),func(db interface{}){
			fmt.Println(db.(*Content).Title)
		})
		if err != nil {
			panic(err)
		}
	})

}




func main (){
	select{}
}

