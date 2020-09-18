package main
import (
	"fmt"
	"io"
	"sort"
	"time"
	"io/ioutil"
	"net/url"
	"net/http"
	"encoding/json"
	"github.com/zaddone/studySystem/request"
	"github.com/lunny/html2md"
	"github.com/gin-gonic/gin"
	"github.com/PuerkitoBio/goquery"
	//"github.com/boltdb/bolt"
	"flag"
	"strings"
	"bytes"
	"regexp"
	//"reflect"
	"ContentServices/content"
)
var (
	Router  = gin.Default()
	searchzhihuUrl *url.URL
	port = flag.String("p","8080","port")
	Sleep = flag.Int("s",600,"port")
	addr = flag.String("a","http://127.0.0.1:8080","addr")

	regG = regexp.MustCompile("解说|福利|色情")
	regM = regexp.MustCompile(`[0-9]+`)
	regS = regexp.MustCompile(`\S+\$\S+\.m3u8`)

	regT *regexp.Regexp = regexp.MustCompile(`[0-9|a-z|A-Z|\p{Han}]+`)
	regK *regexp.Regexp = regexp.MustCompile(`[0-9a-zA-Z]+|\p{Han}`)
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
func NewContentZyw(t,c string,w []string) (co *content.Content) {
	co = &content.Content{
		Title:t,
		Content:c,
		Site:"video",
		Type:2,
		Update:time.Now().Unix(),
		//words:w,
	}
	co.SetWord(w)
	co.SetId(strings.Join(co.GetWords(),""))
	return

}
func getPage_okzyw(url string,h func(interface{}) error )error{

	res,err := http.Get("http://okzyw.com"+url)
	if err != nil {
		return err
	}
	doc,err := goquery.NewDocumentFromReader(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}
	Title := doc.Find(".vodInfo .vodh h2").Text()
	keyMap := map[string]bool{}
	//getTitleKey(Title,func(w string){
	//	keyMap[w]=true
	//})
	ts := regT.FindAllString(Title,-1)
	//self.Title = strings.Join(ts," ")
	for _,l := range ts{
		keyMap[l]=true
	}
	doc.Find(".vodinfobox li span").Each(func(i int,s *goquery.Selection){
		for _,l := range regT.FindAllString(s.Text(),-1){
			kl := regM.ReplaceAllString(l,"")
			if len(kl) ==0 {
				continue
			}
			keyMap[l]=true
		}
	})
	key :=make([]string,0,len(keyMap))
	for k,_:= range keyMap {
		key=append(key,k)
	}
	sort.Strings(key)
	//fmt.Println(self.key)
	tt := doc.Find(".ibox.playBox .vodplayinfo").Text()
	vod := regS.FindAllString(tt,-1)
	if len(vod) == 0 {
		//fmt.Println(tt)
		return fmt.Errorf("find Not vod")
	}
	cont := NewContentZyw(Title,strings.Join(vod,"|"),key)
	return h(cont)

}

func init(){
	flag.Parse()
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

			err = runR()
			if err != nil {
				fmt.Println(err)
			}
			res,err := http.Get(*addr+"/syncwx")
			if err != nil {
				fmt.Println(err)
			}else{
				fmt.Println(res.Status)
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

			if h != nil {
				h(c)
			}
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

	buf,err := json.Marshal(c.ToMap())
	if err != nil {
		return err
	}
	res,err := http.Post(*addr+"/update","application/json",bytes.NewReader(buf))
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf(res.Status)
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
		//fmt.Println(words)
		err :=  searchZhihu(words.(string),func(db interface{}){
			fmt.Println(db.(*content.Content).Title)
		})
		if err != nil {
			panic(err)
		}
	})

}

func getPageList_okzyw(page int,readPage func(string,string)error)error{
	res,err := http.Get(fmt.Sprintf("http://okzyw.com/?m=vod-index-pg-%d.html",page))
	if err != nil {
		return err
	}
	doc,err := goquery.NewDocumentFromReader(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}
	doc.Find(".xing_vb li").EachWithBreak(func(i int,s *goquery.Selection)bool {
		if regG.MatchString(s.Find("span.xing_vb5").Text()){
			//fmt.Println(s.Find("span.xing_vb4").Text())
			return true
		}
		title :=s.Find("span.xing_vb4 a")
		name := regexp.MustCompile(`\s`).ReplaceAllString(title.Text(),"")
		//fmt.Println(name)
		val,ok := title.Attr("href")
		if !ok{
			return true
		}
		//strup := s.Find("span.xing_vb6").Text()
		//if strup=="" {
		//	strup = s.Find("span.xing_vb7").Text()
		//	if strup == "" {
		//		return true
		//	}
		//}
		err = readPage(name,val)
		if err != nil {
			return false
			//if err == io.EOF {
			//	return false
			//}
			fmt.Println(err)
		}
		return true
	})
	return err

}
func runR() error {
	for page:=1;;page++{
		coo := 0
		err := getPageList_okzyw(page,func(name,uri string)error{
			err :=  getPage_okzyw(uri,func(c interface{})error{
				con := c.(*content.Content)
				con.Title = name
				err := searchZhihu(name,nil)
				if err != nil {
					fmt.Println(err)
				}
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

