package main
import (
	"fmt"
	"io"
	"time"
	"io/ioutil"
	"net/url"
	"net/http"
	"encoding/json"
	"github.com/zaddone/studySystem/request"
	"github.com/lunny/html2md"
	"github.com/gin-gonic/gin"
	"flag"
	"strings"
	"bytes"
	"ContentServices/content"
)
var (
	Router  = gin.Default()
	searchzhihuUrl *url.URL
	port = flag.String("p","8080","port")
	Sleep = flag.Int("s",600,"port")
	addr = flag.String("a","http://127.0.0.1:8080","addr")
)
func NewContentZhihu(t,c,a string) (co *content.Content,err error) {
	co = &content.Content{
		Title:t,
		Content:c,
		Author:a,
		Site:"zhihu",
		Type:1,
		Update:time.Now().Unix(),
	}
	err = co.SetWords()
	if err != nil {
		return nil,err
	}
	co.SetId(strings.Join(co.GetWords(),""))
	return
}

func init(){
	//flag.Parse()
	var err error
	searchzhihuUrl,err = url.Parse("https://api.zhihu.com/search_v3?advert_count=0&correction=1&lc_idx=0&limit=20&offset=20&q=%E5%9B%B4%E6%A3%8B&show_all_topics=0&t=general")
	if err != nil {
		panic(err)
	}
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
			err = PostUpdate(c)
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

func PostUpdate(c *content.Content)error{

	var buf bytes.Buffer
	err := json.NewEncoder(buf).Encode(c)
	if err != nil {
		return err
	}
	res,err := http.Post(*addr,"application/json",buf)
	if err != nil {
		return err
	}
	db,err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(db))
	return nil

}

func run() error {
	//fmt.Println("ok")
	err := initZhihu()
	if err != nil {
		return err
	}
	return hotZhihu(func(words interface{}){
		fmt.Println(words)
		err :=  searchZhihu(words.(string),func(db interface{}){
			fmt.Println(db.(*content.Content).Title)
		})
		if err != nil {
			panic(err)
		}
	})

}
func runR() error {
	for page:=1;;page++{
		coo := 0
		err := getPageList(page,func(name,uri string)error{
			err :=  getPage(uri,func(c interface{})error{
				con := c.(*content.Content)
				con.Title = name
				//fmt.Println(con.Title)
				return PostUpdate(con)
			})
			if err == nil {
				coo ++
			}
			return nil
		})
		if err != nil  && err != io.EOF {
			fmt.Println("end",err)
			return err
		}
		if coo==0{
			return nil
		}
		//return err
	}
	return nil


}

func main (){
	select{}
}

