package main
import(
	"net/http"
	"strings"
	"fmt"
	"github.com/boltdb/bolt"
	"sort"
	"io"
	"time"
	"regexp"
	"github.com/PuerkitoBio/goquery"
)
var (
	regG = regexp.MustCompile("解说|福利|色情")
	regM = regexp.MustCompile(`[0-9]+`)
	regS = regexp.MustCompile(`\S+\$\S+\.m3u8`)
	//rootUrl = "http://www.okzyw.com"
)

func getPageList(page int,readPage func(string,string)error)error{
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

func NewContentZyw(t,c string,w []string) (co *Content,err error) {
	co = &Content{
		Title:t,
		Content:c,
		Site:"video",
		Type:2,
		Update:time.Now().Unix(),
		words:w,
	}
	//err = co.setWords()
	//if err != nil {
	//	return nil,err
	//}
	co.setId(strings.Join(co.words,""))
	return

}
func getTitleKey(t string,h func(string))error{

	ts := regK.FindAllString(t,-1)
	key := map[string]bool{}
	for i,k := range ts {
		key[k] = true
		for _,k_ := range ts[(i+1):] {
			k += k_
			key[k]= true
		}
	}
	if len(key) == 0 {
		return nil
	}
	return openDB(wordsFileDB,false,func(t *bolt.Tx)error{
		b:= t.Bucket(wordDB)
		if b == nil {
			return fmt.Errorf("bucket is nil")
		}
		for k,_ := range key {
			v := b.Get([]byte(k))
			if v != nil {
				h(k)
				//ks = append(ks,k)
			}
		}
		return nil
	})


}
func getPage(url string,h func(interface{}) error )error{

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
	cont,err := NewContentZyw(Title,strings.Join(vod,"|"),key)
	if err != nil {
		return err
	}
	return h(cont)

}

func runRead() error {
	for page:=1;;page++{
		coo := 0
		err := getPageList(page,func(name,uri string)error{
			err :=  getPage(uri,func(c interface{})error{
				con := c.(*Content)
				con.Title = name
				//fmt.Println(con.Title)
				err := con.saveWithDB(false,func(c_ *Content,b *bolt.Bucket)error{
					if len(con.Content) != len(c_.Content) {
						fmt.Println(con.Title,c_.Title)
						err := con.savedb(b)
						if err != nil {
							return err
						}
						getTitleKey(con.Title,func(w string){
							con.words = append(con.words,w)
							//keyMap[w]=true
						})
						err = con.addSame()
						if err != nil {
							return err
						}
						return io.EOF
					}else{
					//	return fmt.Errorf("a is same")
					}
					return io.EOF
				})
				if err != nil {
					return err
				}
				fmt.Println(con.Title)

				err = con.addSame()
				if err != nil {
					//return err
					fmt.Println(err)
				}
				if err == nil {
					return con.saveWordsWithDB()
				}
				return err
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
