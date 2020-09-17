package content
import(
	//"time"
	"os"
	"io"
	"sort"
	"fmt"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/gob"
	"bytes"
	"strings"
	"regexp"
	"github.com/boltdb/bolt"
)
var (

	regT *regexp.Regexp = regexp.MustCompile(`[0-9|a-z|A-Z|\p{Han}]+`)
	regK *regexp.Regexp = regexp.MustCompile(`[0-9a-zA-Z]+|\p{Han}`)
	ContentDB = "Page.db"
	wordsFileDB = "words.db"
	linkFileDB = "link.db"
	tmpFileDB = "tmp.db"
	tmpid = []byte("tmpid")
	tmpWords = []byte("tmpWords")
	tmplink = []byte("tmplink")

	pageDB = []byte("list")
	wordDB = []byte("word")

	parentDB = []byte("parent")
	chidrenDB = []byte("chidren")

)

type sobj struct {
	id string
	n float64
}

func sortSobj(li []sobj,i int){

	if i <= 0 {
		return
	}
	I:=i-1
	if li[I].n<li[i].n {
		li[I],li[i] = li[i],li[I]
		sortSobj(li,I)
	}

}

type Content struct {
	Title string
	Content string
	Author string
	Site string
	Update int64
	Type int
	id  []byte
	words []string
	parentId []byte
	children []byte
}
func (self *Content) GetWords()[]string{
	return self.words
}
func stringsSort(s []string,i int){

	if i == 0 {
		return
	}
	I := i-1
	if len(s[i]) > len(s[I]) {
		s[i],s[I] = s[I],s[i]
		stringsSort(s,I)
	}

}
func SearchWithWords(k string,searchMax int,h func(interface{})) error {
	return searchWithWords(k,searchMax,h)
}

func searchWithWords(k string,searchMax int,h func(interface{})) error {

	key := []string{}
	for _,t := range regT.FindAllString(k,-1){
		if len(t)<=1 {
			continue
		}
		lr := regK.FindAllString(t,-1)
		for j:=0;j<len(lr);j++{
			for _j:=j+1;_j<=len(lr);_j++ {
				k :=strings.ToLower(strings.Join(lr[j:_j],""))
				if len([]rune(k))>1{
					ik := len(key)
					key = append(key,k)
					stringsSort(key,ik)
				}
			}
		}
	}
	fmt.Println(key)

	ids := map[string]float64{}
	//var keys []string
	err := openDB(wordsFileDB,false,func(t *bolt.Tx)error{
		b:= t.Bucket(wordDB)
		if b == nil {
			return fmt.Errorf("bucket is nil")
		}
		for _i,k := range key {
			if len(k) == 0 {
				continue
			}
			v := b.Get([]byte(k))
			if v == nil {
				continue
			}
			for _j := _i+1; _j<len(key); _j++{
				if strings.Contains(k,key[_j]){
					key[_j]=""
				}
			}

			//keys = append(keys,k)
			le := float64(len(v))
			v__ := le/16
			var i float64
			for i =0;i<le;i+=16 {
				I := int(i)
				//ids[fmt.Sprintf("%x",v[I:I+16])] += v__ + (le-i)/le
				ids[string(v[I:(I+16)])] += v__ + (le-i)/le
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("not words key")
	}
	//fmt.Println(keys)
	getMin := func (m map[string]float64) string {
		var minid string
		var minv float64
		for k,v := range m {
			if v<minv || minv == 0 {
				minv = v
				minid = k
			}
		}
		delete(m,minid)
		//fmt.Println(minv)
		return minid
	}
	lm := float64(len(ids))
	mlist := make(map[string]float64)
	err = openDB(linkFileDB,false,func(t *bolt.Tx)error{
		b := t.Bucket(parentDB)
		if b == nil {
			return fmt.Errorf("parent is nil")
		}
		b_ := t.Bucket(chidrenDB)
		if b == nil {
			return fmt.Errorf("chidren is nil")
		}
		for i:=lm;len(ids)>0;i-- {
			id_ := getMin(ids)
			mlist[id_] +=i
			//par := b.Get(id_)
			//if len(par) >0 {
			//	mlist[string(par)] +=1
			//}
			chi := b_.Get([]byte(id_))
			chilen := len(chi)
			if chilen >0 {
				chilen_ := float64(chilen)
				for I:=0;I<chilen;{
					I_ := I + 16
					mlist[string(chi[I:I_])] += float64(I_)/chilen_
					I = I_
				}
			}
			if len(mlist)>searchMax {
				return nil
			}

		}
		return nil
	})
	if err != nil {
		return err
	}
	lmm := len(mlist)
	if lmm == 0 {
		return fmt.Errorf("not words key")
	}

	intli :=make([]sobj,0,lmm)
	for k,v := range mlist {
		le := len(intli)
		intli = append(intli,sobj{k,v})
		sortSobj(intli,le)
	}
	//fmt.Println(intli)
	//objlist := make([]interface{},0,len(mlist))
	return openDB(ContentDB,false,func(t *bolt.Tx)error{
		b := t.Bucket(pageDB)
		if b == nil {
			return fmt.Errorf("page is nil")
		}
		for _,l := range intli {
			c := &Content{
				id:[]byte(l.id),
			}
			fmt.Println(l.n)
			e := c.Load(b.Get(c.id))
			if e != nil {
				fmt.Println(e)
				continue
				//panic(e)
			}
			h(c)
		}
		return nil
	})
}

func hexToByte(s string)[]byte{
	db,err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return db
}

//func NewContentZhihu(t,c,a string) (co *Content,err error) {
//	co = &Content{
//		Title:t,
//		Content:c,
//		Author:a,
//		Site:"zhihu",
//		Type:1,
//		Update:time.Now().Unix(),
//	}
//	err = co.setWords()
//	if err != nil {
//		return nil,err
//	}
//	co.setId(strings.Join(co.words,""))
//	return
//}
func GetWordsKey(t string,h func(string))error{
	return getWordsKey(t,h)
}
func getWordsKey(t string,h func(string))error{

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

func openDB(name string,writable bool,h func(*bolt.Tx)error)error{
	db,err := bolt.Open(name,0600,nil)
	if err != nil {
		return err
	}
	defer db.Close()
	t,err := db.Begin(writable)
	if err != nil {
		return err
	}
	if writable {
		defer t.Commit()
	}
	return h(t)
}
func (self *Content) SaveWordsWithDB() error {
	return self.saveWordsWithDB()
}

func (self *Content) saveWordsWithDB() error {

	return openDB(wordsFileDB,true,func(t *bolt.Tx)error{
		b_,err := t.CreateBucketIfNotExists(wordDB)
		if err != nil {
			return err
		}
		for _,w := range self.words {
			kw := []byte(w)
			v := b_.Get(kw)
			if v == nil {
				b_.Put(kw,self.id)
			} else {
				v = append(v,self.id...)
				if len(v)/16 > 100 {
					v = v[(len(v) - 1600):]
				}
				b_.Put(kw,v)
			}
		}
		return nil
	})

}
func (self *Content) SaveWithDB(isupdate bool,h func(*Content,*bolt.Bucket)error) error {
	return self.saveWithDB(isupdate,h)
}

func (self *Content) saveWithDB(isupdate bool,h func(*Content,*bolt.Bucket)error) error {
	if len(self.id)==0  {
		return fmt.Errorf("id is nil")
	}
	return openDB(ContentDB,true,func(t *bolt.Tx)error{
		b,err := t.CreateBucketIfNotExists(pageDB)
		if err != nil {
			return err
		}
		if !isupdate {
			v := b.Get(self.id)
			if v != nil {
				if h != nil {
					con := &Content{id:self.id}
					err := con.Load(v)
					if err != nil {
						panic(err)
					}
					return h(con,b)
				}
				return io.EOF
				//return fmt.Errorf("is same")
			}
		}
		return self.savedb(b)

	})
}
func (self *Content) Savedb(b *bolt.Bucket)error{
	return self.savedb(b)
}
func (self *Content) savedb(b *bolt.Bucket)error{
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(self)
	if err != nil {
		return err
	}
	return  b.Put(self.id,buf.Bytes())
}

func (self *Content) LoadWithDB(id []byte)error{
	self.id = id
	err := openDB(ContentDB,false,func(t *bolt.Tx)error{
		b := t.Bucket(pageDB)
		if b == nil {
			return fmt.Errorf("bucket is nil")
		}
		v := b.Get(id)
		if v  == nil {
			return fmt.Errorf("found not with id")
		}
		return self.Load(v)
	})
	if err != nil {
		return err
	}
	return openDB(linkFileDB,false,func(t *bolt.Tx)error{
		b := t.Bucket(parentDB)
		if b != nil {
			self.parentId = b.Get(self.id)
		}
		b = t.Bucket(chidrenDB)
		if b != nil {
			self.children = b.Get(self.id)
		}
		return nil
	})
}
func (self *Content) Load(db []byte)error{
	return gob.NewDecoder(bytes.NewBuffer(db)).Decode(self)
}
func (self *Content) showId() string {
	return fmt.Sprintf("%x",self.id)
}
func (self *Content) SetId (db string){
	//self.id = fmt.Sprintf("%x",md5.Sum([]byte(db)))
	id := md5.Sum([]byte(db))
	self.id = id[:]
}
func (self *Content) AddSame() error {
	return self.addSame()
}
func (self *Content) addSame() error {

	ids := map[string]float64{}
	id__ := string(self.id)
	err := openDB(wordsFileDB,false,func(t *bolt.Tx)error{
		b := t.Bucket(wordDB)
		if b == nil {
			return fmt.Errorf("bucket is nil")
		}
		for _,w := range self.words {
			v := b.Get([]byte(w))
			if v == nil {
				continue
			}
			le := float64(len(v))
			v__ := (le/16)
			var i float64
			for i=0;i<le;i+=16 {
				I := int(i)
				//ids[fmt.Sprintf("%x",v[I:I+16])] += v__ + (le-i)/le
				__id := string(v[I:I+16])
				if strings.EqualFold(id__,__id){
					continue
				}
				ids[__id] += v__ + (le-i)/le
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	var minid string
	var minv float64
	for k,v := range ids {
		if v<minv || minv == 0 {
			minv = v
			minid = k
		}
		//hex.DecodeString()
	}
	self.parentId = []byte(minid)
	return openDB(linkFileDB,true,func(t *bolt.Tx)error{
		b,err := t.CreateBucketIfNotExists(parentDB)
		if err != nil {
			return err
		}
		err = b.Put(self.id,self.parentId)
		if err != nil {
			return err
		}
		b_,err := t.CreateBucketIfNotExists(chidrenDB)

		if err != nil {
			return err
		}
		v := b_.Get(self.parentId)
		if v == nil {
			return b_.Put(self.parentId,self.id)
		}else{
			v = append(v,self.id...)
			if len(v)/16 > 100 {
				v = v[(len(v) - 1600):]
			}
			return b_.Put(self.parentId,v)
		}
	})

}

func ClearTmpDB() (err error) {
	return os.Remove(tmpFileDB)
}

func ReadAllDB(conh,wordh,child func(io.Reader))error{
	var cons,words,ch  [][]byte

	err := openDB(linkFileDB,false,func(lt *bolt.Tx)error{
		pb := lt.Bucket(parentDB)
		if pb == nil {
			return nil
		}
		cb := lt.Bucket(chidrenDB)
		if cb == nil {
			return nil
		}
		err :=  cb.ForEach(func(k,v []byte)error{
			mdb := map[string]interface{}{}
			mdb["_id"] = fmt.Sprintf("%x",k)
			str:=make([]string,0,len(v)/16)
			var i,I int
			for i=0;i<len(v);i = I{
				I = i+16
				str = append(str,fmt.Sprintf("%x",v[i:I]))
			}
			mdb["link"] = str
			db_,err := json.Marshal(mdb)
			if err != nil {
				return err
			}
			ch = append(ch,db_)
			return nil
		})
		if err != nil {
			return err
		}
		if len(ch)>0{
			child(bytes.NewReader(bytes.Join(ch,[]byte{'\n'})))
		}
		return openDB(ContentDB,false,func(t_ *bolt.Tx)error{
		bc := t_.Bucket(pageDB)
		if bc == nil {
			return nil
		}

		err = bc.ForEach(func(k,v []byte)error{
			con := &Content{id:k}
			con.Load(v)
			con.parentId = pb.Get(k)
			db,err := json.Marshal(con.ToMapWX())
			if err != nil {
				return err
			}
			cons = append(cons,db)
			return nil

			//children := cb.Get(k)

		})
		if err != nil {
			return err
		}
		if len(cons)>0{
		conh(bytes.NewReader(bytes.Join(cons,[]byte{'\n'})))
		}
		return nil

		})
	})
	if err != nil {
		return err
	}
	return  openDB(wordsFileDB,false,func(t_ *bolt.Tx)error{
		b_ := t_.Bucket(wordDB)
		if b_ == nil {
			return nil
		}
		err := b_.ForEach(func(k,v []byte)error{
			mdb := map[string]interface{}{}
			mdb["_id"] = string(k)
			str:=make([]string,0,len(v)/16)
			var i,I int
			for i=0;i<len(v);i = I{
				I = i+16
				str = append(str,fmt.Sprintf("%x",v[i:I]))
			}
			mdb["link"] = str
			db,err := json.Marshal(mdb)
			if err != nil {
				return err
			}
			words = append(words,db)
			return nil
		})
		if err != nil {
			return err
		}
		if len(words)>0{
			wordh(bytes.NewReader(bytes.Join(words,[]byte{'\n'})))
		}
		return nil

	})
}

func ReadTmpAll(conh,wordh,child func(io.Reader))error{
	var cons,words,ch  [][]byte
	return openDB(tmpFileDB,false,func(t *bolt.Tx)error{
		b := t.Bucket(tmpid)
		if b == nil {
			return fmt.Errorf("tmpid is nil")
		}
		err :=  openDB(ContentDB,false,func(t_ *bolt.Tx)error{
			b_ := t_.Bucket(pageDB)
			if b_ == nil {
				return nil
			}
			return openDB(linkFileDB,false,func(lt *bolt.Tx)error{
			lb := lt.Bucket(chidrenDB)
			if lb == nil {
				return nil
			}
			return b.ForEach(func(k,_v []byte)error{
				v := b_.Get(k)
				con := &Content{id:k,parentId:_v}
				con.Load(v)
				db,err := json.Marshal(con.ToMapWX())
				if err != nil {
					return err
				}
				cons = append(cons,db)
				lv := lb.Get(_v)
				if lv == nil {
					return nil
				}
				mdb := map[string]interface{}{}
				mdb["_id"] = fmt.Sprintf("%x",_v)
				str:=make([]string,0,len(lv)/16)
				var i,I int
				for i=0;i<len(lv);i = I{
					I = i+16
					str = append(str,fmt.Sprintf("%x",lv[i:I]))
				}
				mdb["link"] = str
				db_,err := json.Marshal(mdb)
				if err != nil {
					return err
				}
				ch = append(ch,db_)


				return nil
			})
			})
		})
		if err != nil {
			return err
		}
		if len(cons)>0{
		conh(bytes.NewReader(bytes.Join(cons,[]byte{'\n'})))
		}
		if len(ch)>0{
		child(bytes.NewReader(bytes.Join(ch,[]byte{'\n'})))
		}
		//return nil
		_b := t.Bucket(tmpWords)
		if _b == nil {
			return fmt.Errorf("words is nil")
			return nil
		}
		err = openDB(wordsFileDB,false,func(t_ *bolt.Tx)error{
			b_ := t_.Bucket(wordDB)
			if b_ == nil {
				return nil
			}
			return _b.ForEach(func(k,_ []byte)error{
				v := b_.Get(k)
				mdb := map[string]interface{}{}
				mdb["_id"] = string(k)
				str:=make([]string,0,len(v)/16)
				var i,I int
				for i=0;i<len(v);i = I{
					I = i+16
					str = append(str,fmt.Sprintf("%x",v[i:I]))
				}
				mdb["link"] = str
				db,err := json.Marshal(mdb)
				if err != nil {
					return err
				}
				words = append(words,db)
				//wordh(db)
				return nil
			})
		})
		if err != nil {
			return err
		}
		if len(words)>0{
			wordh(bytes.NewReader(bytes.Join(words,[]byte{'\n'})))
		}
		return nil
	})

}
func (self *Content) UpdateTmp() (err error) {
	return openDB(tmpFileDB,true,func(t *bolt.Tx)error{
		b,err := t.CreateBucketIfNotExists(tmpid)
		if err != nil {
			return err
		}
		if len(self.parentId)==0 {
			self.parentId = []byte{0}
		}
		err = b.Put(self.id,self.parentId)
		if err != nil {
			return err
		}

		b,err = t.CreateBucketIfNotExists(tmpWords)
		for _,w := range self.words {
			err = b.Put([]byte(w),[]byte{'0'})
			if err != nil {
				fmt.Println(err)
			}
		}
		return nil
	})
}

func (self *Content) UpdateInfo() (err error) {
	if self.Type == 2 {
		err = self.SaveWithDB(false,func(c_ *Content,b *bolt.Bucket)error{
			if strings.EqualFold(self.Content,c_.Content) {
				return fmt.Errorf("is same")
			}
			fmt.Println(self.Title,c_.Title)
			err := self.Savedb(b)
			if err != nil {
				return err
			}
			GetWordsKey(self.Title,func(w string){
				self.SetWord(append(self.GetWords(),w))
			})
			err = self.AddSame()
			if err != nil {
				return err
			}
			err = self.SaveWordsWithDB()
			if err != nil {
				return err
			}
			return io.EOF
		})
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

	}else{
		err = self.SaveWithDB(false,nil)
		if err != nil {
			return err
		}
	}
	err = self.AddSame()
	if err != nil {
		fmt.Println(err)
		//return err
	}
	return self.SaveWordsWithDB()

}
func (self *Content) SetWord(w []string) {
	self.words = w
}
func (self *Content) SetWords() error {

	if self.Title == "" {
		return fmt.Errorf("title is nil")
	}
	if self.Content == ""{
		return fmt.Errorf("title is nil")
	}
	//titl := regT.FindAllString(self.Title,-1)
	//newTi := make([]string,0,len(titl))
	var newTi []string
	for _,t := range regT.FindAllString(self.Title,-1){
		if len(t)>2 {
			newTi = append(newTi,t)
		}
	}
	//self.Content = clearHerf(self.Content)
	for _,t := range regT.FindAllString(clearHerf(self.Content),-1){
		if len(t)>2 {
			newTi = append(newTi,t)
		}
	}
	self.words = split_(newTi)
	return nil

}
func (self *Content) ToMapWX()(m map[string]interface{}) {
	m = map[string]interface{}{}
	m["Title"] = self.Title
	m["Content"] = self.Content
	m["Author"] = self.Author
	m["Site"] = self.Site
	m["Update"] = self.Update
	m["Type"] = self.Type
	m["_id"] = self.showId()
	//m["words"] = self.words
	m["parentId"] = fmt.Sprintf("%x",self.parentId)
	//m["children"] = self.children
	return m

}

func (self *Content) ToMap()(m map[string]interface{}) {
	m = map[string]interface{}{}
	m["Title"] = self.Title
	m["Content"] = self.Content
	m["Author"] = self.Author
	m["Site"] = self.Site
	m["Update"] = self.Update
	m["Type"] = self.Type
	m["id"] = self.id
	m["words"] = self.words
	m["parentId"] = self.parentId
	m["children"] = self.children
	return m

}

func clearHerf(db string) string {

	return regexp.MustCompile(`\[[^\]]+\]\([^\)]+\)`).ReplaceAllStringFunc(db,func(b string)string{
		//if strings.Contains(b,"image") {
		//	return b
		//}
		//fmt.Println(b)
		return ""
	})

}

func split_(li []string) []string {

	key := map[string]int{}
	for _,l := range li {
		lr := regK.FindAllString(l,-1)
		lenl := len(lr)
		for j:=0; j<lenl; j++{
			_j := j+1
			if _j<lenl {
				if lr[j] == lr[_j]{
					continue
				}
			}
			for ; _j<=lenl; _j++ {
				k :=strings.ToLower(strings.Join(lr[j:_j],""))
				if len([]rune(k))>1{
					key[k]+=1
				}
			}
		}
	}
	var lkey,llkey []string
	//nkey := map[string]int{}
	for k,v := range key {
		if v<=1 {
			//delete(key,k)
			continue
		}
		//nkey[k] = v
		lkey = append(lkey,k)
		//fmt.Println(k,v)
	}
	sort.Strings(lkey)
	//fmt.Println(key,lkey)
	G:
	for _,k := range lkey {
		//delete(key,k)
		for _,_k := range lkey {
			if len(k) >= len(_k) {
				//fmt.Println(k,_k)
				continue
			}
			if strings.Contains(_k,k) && (key[k]==key[_k]) {
				//fmt.Println(_k,k)
				//delete(nkey,k)
				continue G
			}
		}
		llkey = append(llkey,k)
	}
	//fmt.Println(len(key))
	//for k,v := range nkey{
	//	llkey = append(llkey,k)
	//	//fmt.Println(k,v)
	//}
	//sort.Strings(llkey)
	return llkey

}


