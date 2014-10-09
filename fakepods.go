/*

Doesn't do concurrency properly, I think.  We probably need locks of
some sort, or goroutines, on each of the main structures -- the
Cluster, the Pod, and the Resource.  What happens when someone is
slowly PUT'ing new bytes to a resource when someone else is reading
it?



*/

package main

import (
	"bytes"
	"net/http"
	//"net/url"
	"log"
	"fmt"
	"io"
	"strings"
	//"time"
	"regexp"
	//"errors"
	"encoding/json"
	"strconv"
	"sort"
	"flag"
	"os"
	"time"
	
)

type Resource struct {
	ContentType string
	Body bytes.Buffer
	Data map[string]interface{}
	LastMod uint64
}

func (res* Resource) UpdateData() {
	res.Data["_etag"] = res.LastMod
}

type ById []jsonobj
func (a ById) Len() int { return len(a) }
func (a ById) Swap(i, j int)  { a[i], a[j] = a[j], a[i] }
func (a ById) Less(i, j int) bool { return a[i]["_id"].(string) < a[j]["_id"].(string) }

type Pod struct {
	URL string
	Resources map[string]*Resource
	NextVersion uint64
	ResourceCounter uint64
}

func NewPod(podURL string) *Pod {
	pod := &Pod{podURL, make(map[string]*Resource),	0, 0}
	pods[podURL] = pod
	pod.Resources = make(map[string]*Resource)
	log.Printf("created pod %q\n", pod)
	return pod
}

func StreamToString(stream io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.String()
}

var pods = make(map[string]*Pod)
var version = uint64(0)

func main() {
	
	var port = flag.String("port", "8080", "web server port")
	var logdir = flag.String("logdir", "/var/log/fakepods", "where to put the log files")
	var dolog = flag.Bool("log", false, "log to file instead of stdout")
	var restore = flag.String("restore", "", "restore state from given json dump file")
	flag.Parse()

	if *dolog {
		err := os.MkdirAll(*logdir, 0700)
		if err != nil {
			panic(err)
		}
		logfilename := *logdir+"/log-"+time.Now().Format("20060102-030405")
		fmt.Printf("logging to %s\n", logfilename)
		logfile, err := os.Create(logfilename)
		if err != nil {
			// not sure why I'm getting "No such file or directory" 
			// panic(err)
		}
		log.SetOutput(logfile)
	}

	if *restore != "" {
		fi, err := os.Open(*restore)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		restoreCluster(fi)
	}

    http.HandleFunc("/", homeHandler)

	log.Printf("server started, listening on port %s", *port)
	err := http.ListenAndServe(":"+*port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

var validPodname *regexp.Regexp

func init() {
	validPodname = regexp.MustCompile("^[a-z][a-z0-9]*$")
}


func homeHandler(w http.ResponseWriter, r *http.Request) {

	/*
       Figure out which URL pattern is being used, and
       take the URL apart.   We support both

          http://PODNAME.host.domain[/PATH]
       and
          http://host.domain/pod/PODNAME[/PATH]

    */
	log.Printf("Request %q\n", r)

	var podURL, podname, path string
	pathparts:=strings.Split(r.URL.Path, "/")
	if pathparts[1] == "pod" {
		if len(pathparts) > 2 {
			podname = pathparts[2]
			podURL = "http://"+r.Host + "/pod/" + podname
			path = strings.Join(pathparts[3:], "/")
		}
	} else {
		hostparts := strings.Split(r.Host, ".")
		// hardcoding that foo.bar or www.* is the non-pod panel (!)
		if len(hostparts) > 2 && hostparts[0] != "www" {
			podname = hostparts[0]
			podURL = "http://"+r.Host
		}
		path = r.URL.Path[1:]		
	}
	if podname != "" && !validPodname.MatchString(podname) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Pod names must match regexp %s\n", validPodname)
		return
	}


	log.Printf("\n")
	log.Printf("PodURL  %q", podURL)
	log.Printf("Podname %q\n", podname)
	log.Printf("Path    %q\n", path)

	// we's like the query parameters parsed, but we don't want the 
	// POST'd body consumed, even if it's application/x-www-form-urlencoded


	if r.Method == "GET" { 
		r.ParseForm() 
		log.Printf("Args    %q\n", r.Form)
	}

	// long poll 
	if r.Method == "GET" { 
		var err error
		var val string
		var waitForVersionAfter uint64

		if val = r.Header.Get("Wait-For-None-Match"); val != "" {
			log.Printf("wait-for-version-after  %q\n", val)
			waitForVersionAfter,err = strconv.ParseUint(val, 10, 64)
			if err != nil {
				log.Println("converting Wait-For-None-Match:", err)
			}
			if version == 0 && waitForVersionAfter != 0 {
				// server must have just restarted; don't wait
				// this time
			} else {
				if waitForVersionAfter >= version {
					pauseForChanges()
				} else {
					log.Printf("Wait-For-None-Match already satisfied %d, at %d",
						waitForVersionAfter, version);
				}
			}
		}
	}

	


	var pod *Pod
	var res *Resource

	pod = pods[podURL]
	//log.Printf("Pod     %q\n", pod)

	if pod!=nil && path!="" {
		res = pod.Resources[path]
	}
		
	log.Printf("Checking origin\n");
	if origin := r.Header.Get("Origin"); origin != "" {
		log.Printf("Allowing access from origin: %q\n", origin)
        w.Header().Set("Access-Control-Allow-Origin", origin)
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE, PATCH")
        w.Header().Set("Access-Control-Expose-Headers",
			"Location, ETag")
        w.Header().Set("Access-Control-Allow-Headers",
            "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Wait-For-None-Match")
    }

	log.Printf("Method  %q\n", r.Method)
	switch r.Method {
	case "DELETE":
		if res == nil {
		}
		// @@ IMPLEMENT
	case "GET":
		switch path {
		case "_trace": 
			// launch a goroutine that copies the recent and ongoing
			// log entries
		case "":
			if podname != "" {
				if pod == nil { http.NotFound(w,r); return }
				obj:=jsonobj{
					//"_type": "Pod",
					"_id":podURL,
					"resourcesCreated":pod.ResourceCounter,
				}
				offerJSON(w,r,obj)
			} else {
				items := make([]interface{},0)
				for podURL, pod:= range pods {
					obj:=jsonobj{
						"_type": "pod",
						"_id":podURL,
						"resourcesCreated":pod.ResourceCounter,
					}
					items = append(items, obj)
				}
				frame:=jsonobj{/*"_type":"PodCluster",*/ "pods":items}
				offerJSON(w,r,frame)
			}
		case "_active":
			if pod == nil { http.NotFound(w,r); return }
			items := make([]interface{},0)
			for path, res := range pod.Resources {
				if res.Data != nil {
					res.Data["_owner"] = podURL
					res.Data["_id"] = podURL+"/"+path
					res.Data["_etag"] = res.LastMod
					items = append(items, res.Data)
				}
			}
			offerJSON(w,r,jsonobj{"_etag":version,"_members":items})
		case "_nearby":
			items := make([]jsonobj,0)
			for podURL, pod := range pods {
				for path, res := range pod.Resources {
					if res.Data != nil {
						res.Data["_owner"] = podURL
						res.Data["_id"] = podURL+"/"+path
						res.Data["_etag"] = res.LastMod
						items = append(items, res.Data)
					} // else it's non JSON...
				}
			}
			sort.Sort(ById(items))
			offerJSON(w,r,jsonobj{"_etag":version,"_members":items})
		default:
			if res == nil { 
				http.NotFound(w,r) 
				return
			}
			w.Header().Set("Content-Type", res.ContentType)

			if res.Data != nil {
				res.UpdateData()
				bytes, _ := json.MarshalIndent(res.Data, "", "    ")
				w.Write(bytes)
				fmt.Fprintf(w, "\n")
			} else {
				_,_ = res.Body.WriteTo(w)
			}
		}
	case "HEAD": 
		// this is oddly handled by go.   hrm.
	case "CRASH":
		panic("just testing")
	case "OPTIONS": 
		// needed for CORS pre-flight
		return
	case "PATCH":
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Sorry, not implemented yet\n")
	case "POST":
		if path != "" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "You can only post at the root of the pod\n")
			return
		}
		if pod == nil { pod = NewPod(podURL) }
		res = new(Resource)
		res.ContentType = r.Header["Content-Type"][0]
		log.Printf("Content type was %q", res.ContentType)
		if semi := strings.Index(res.ContentType, ";"); semi>0 {
			res.ContentType = res.ContentType[0:semi]
		}
		log.Printf("Content type was %q", res.ContentType)
		res.Body.ReadFrom(r.Body)
		log.Printf("Body was %q", res.Body)
		res.LastMod = pod.NextVersion
		pod.NextVersion++
		name := fmt.Sprintf("r%d", pod.ResourceCounter)
		pod.ResourceCounter++
		pod.Resources[name] = res
		changeWasMade()

		location := podURL+"/"+name
		log.Printf("Location assigned: %q", location)
		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusCreated)

		// try parsing?!
		if res.ContentType == "application/json" || res.ContentType == "application/x-www-form-urlencoded" {
			log.Printf("Parsing JSON %q\n", res.Body.String())
			err := json.Unmarshal(res.Body.Bytes(), &res.Data)
			if err != nil {
				log.Println("error:", err)
			}
			log.Printf("%+v", res.Data)
		}
	case "PUT":
		if res == nil {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Use POST to the pod URL to create, please\n")
			return
		}
		res.ContentType = r.Header["Content-Type"][0]
		log.Printf("Content type was %q", res.ContentType)
		if semi := strings.Index(res.ContentType, ";"); semi>0 {
			res.ContentType = res.ContentType[0:semi]
		}
		log.Printf("Content type was %q", res.ContentType)
		newBody := bytes.Buffer{}
		newBody.ReadFrom(r.Body)
		res.Body = newBody
		log.Printf("Body was %q", res.Body)
		
		res.LastMod++
		pod.NextVersion++

		if res.ContentType == "application/json" || res.ContentType == "application/x-www-form-urlencoded" {
			log.Printf("Parsing JSON %q\n", res.Body.String())
			err := json.Unmarshal(res.Body.Bytes(), &res.Data)
			if err != nil {
				log.Println("error:", err)
			}
			log.Printf("%+v", res.Data)
		}
		changeWasMade()
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
    w.WriteHeader(status)
    if status == http.StatusNotFound {
        fmt.Fprint(w, "custom 404")
    }
}

type jsonobj map[string]interface{}
	
type jsonarr []interface{}

func offerJSON(w http.ResponseWriter, r *http.Request, frame jsonobj) {

	// if they'd prefer HTML, maybe format it as HTML or something?

	bytes, _ := json.MarshalIndent(frame, "", "    ")
	w.Write(bytes)
	fmt.Fprintf(w, "\n")
}

var chch = make(chan chan bool, 1000)
func pauseForChanges() {
	ch := make(chan bool)
	chch <- ch // queue up ch as a response point for us
	_ = <- ch // wait for that response
	return
}

func changeWasMade() {
	// later on, maybe we do the change here, in a single goroutine?
	// resources are created, updated, or deleted
	version++
	
	// go through all of chch and notify them all
	var ch chan bool
	for {
		select {
		case ch = <- chch:
			ch <- true
		default:
			return
		}
	}
}

func restoreCluster(src io.Reader) {
	dec := json.NewDecoder(src)
	var v jsonobj
	if err := dec.Decode(&v); err != nil {
		log.Println(err)
		return
	}
	//var err error
	version = uint64(v["_etag"].(float64))
	log.Printf("etag %d", version)
	//var members []jsonobj
	members := v["_members"].([]interface{})
	//log.Printf("members: %d %q", len(members), members)
	for i,obj := range members {
		log.Printf("member %d: %q\n\n", i, obj)

		// only works for JSON for now
		res := new(Resource)
		res.ContentType = "application/json"
		res.Data = jsonobj(obj.(map[string]interface {}))
		res.LastMod = uint64(res.Data["_etag"].(float64))

		// lookup the _owner pod
		podURL := res.Data["_owner"].(string)
		pod, podExists := pods[podURL]
		if !podExists {
			pod = NewPod(podURL)
			pods[podURL] = pod
		}

		// find the .name   -- will mess up if clster wasnt empty
		name := res.Data["_id"].(string)[len(podURL)+1:]
		pod.ResourceCounter++
		pod.Resources[name] = res
		
		log.Printf("PodURL  %q\n", podURL)
		log.Printf("exists  %q\n", podExists)
		log.Printf("nameL  %q\n", name)

		// maybe delete extra _foo items from res.Data ?
	}
	
}
