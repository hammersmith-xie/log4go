// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package log4go

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
	"errors"
)

type xmlProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type xmlFilter struct {
	Enabled  string        `xml:"enabled,attr"`
	Tag      string        `xml:"tag"`
	Level    string        `xml:"level"`
	Type     string        `xml:"type"`
	FilePath string        `xml:"filepath"`
	Property []xmlProperty `xml:"property"`
}

type xmlLoggerConfig struct {
	Filter []xmlFilter `xml:"filter"`
}

// Load XML configuration; see examples/example.xml for documentation
func (log Logger) LoadConfiguration(filename string) {
	log.Close()
	// Open the configuration file
	fd, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not open %q for reading: %s\n", filename, err)
		os.Exit(1)
	}
	contents, err := ioutil.ReadAll(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not read %q: %s\n", filename, err)
		os.Exit(1)
	}
	xc := new(xmlLoggerConfig)
	if err := xml.Unmarshal(contents, xc); err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not parse XML configuration in %q: %s\n", filename, err)
		os.Exit(1)
	}

	for _, xmlfilt := range xc.Filter {
		var filt LogWriter
		var lvl Level
		bad, good, enabled := false, true, false

		// Check required children
		if len(xmlfilt.Enabled) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required attribute %s for filter missing in %s\n", "enabled", filename)
			bad = true
		} else {
			enabled = xmlfilt.Enabled != "false"
		}
		if len(xmlfilt.Tag) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "tag", filename)
			bad = true
		}
		if len(xmlfilt.Type) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "type", filename)
			bad = true
		}
		if len(xmlfilt.Level) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "level", filename)
			bad = true
		}
		if len(xmlfilt.FilePath) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "filepath", filename)
			bad = true
		}

		switch xmlfilt.Level {
		case "FINEST":
			lvl = FINEST
		case "FINE":
			lvl = FINE
		case "DEBUG":
			lvl = DEBUG
		case "TRACE":
			lvl = TRACE
		case "INFO":
			lvl = INFO
		case "WARNING":
			lvl = WARNING
		case "ERROR":
			lvl = ERROR
		case "CRITICAL":
			lvl = CRITICAL
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter has unknown value in %s: %s\n", "level", filename, xmlfilt.Level)
			bad = true
		}

		// Just so all of the required attributes are errored at the same time if missing
		if bad {
			os.Exit(1)
		}
		var file string
		switch xmlfilt.Type {
		case "file":
			filt, good, file = xmlToFileLogWriter(filename,xmlfilt.FilePath, xmlfilt.Property, enabled)
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not load XML configuration in %s: unknown filter type \"%s\"\n", filename, xmlfilt.Type)
			os.Exit(1)
		}

		// Just so all of the required params are errored at the same time if wrong
		if !good {
			os.Exit(1)
		}

		// If we're disabled (syntax and correctness checks only), don't add to logger
		if !enabled {
			continue
		}
		log[file] = &Filter{lvl, filt}
	}
}

// Parse a number with K/M/G suffixes based on thousands (1000) or 2^10 (1024)
func strToNumSuffix(str string, mult int) int {
	num := 1
	if len(str) > 1 {
		switch str[len(str)-1] {
		case 'G', 'g':
			num *= mult
			fallthrough
		case 'M', 'm':
			num *= mult
			fallthrough
		case 'K', 'k':
			num *= mult
			str = str[0 : len(str)-1]
		}
	}
	parsed, _ := strconv.Atoi(str)
	return parsed * num
}
func xmlToFileLogWriter(filename string, filepath string,props []xmlProperty, enabled bool) (*FileLogWriter, bool, string) {
	file := ""
	format := "[%D %T] [%L] (%S) %M"
	maxlines := 0
	maxsize := 0
	daily := false
	rotate := false

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "filename":
			file = strings.Trim(prop.Value, " \r\n")
			file = filepath +file
		case "format":
			format = strings.Trim(prop.Value, " \r\n")
		case "maxlines":
			maxlines = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1000)
		case "maxsize":
			maxsize = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1024)
		case "daily":
			daily = strings.Trim(prop.Value, " \r\n") != "false"
		case "rotate":
			rotate = strings.Trim(prop.Value, " \r\n") != "false"
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Warning: Unknown property \"%s\" for file filter in %s\n", prop.Name, filename)
		}
	}

	// Check properties
	if len(file) == 0 {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n", "filename", filename)
		return nil, false, file
	}

	// If it's disabled, we're just checking syntax
	if !enabled {
		return nil, true, file
	}
	flw := NewFileLogWriter(file, rotate,true)
	fmt.Println("file:[%s]",file)
	flw.SetFormat(format)
	flw.SetRotateLines(maxlines)
	flw.SetRotateSize(maxsize)
	flw.SetRotateDaily(daily)
	return flw, true, file
}



func (log Logger)initFileLogWriter(name string, path string,islog bool)(error){
	filename:="test"
	filepath:=""
	if len(name) != 0{
		filename=name
	}
	if len(path) != 0{
		filepath=path
	}

	strpath:=&[]string{}
	splitpath(filepath,strpath)
	for _,v:=range *strpath{
		fmt.Printf("%s\n",v)
	}
	CreateDir(*strpath)


	var filt LogWriter
	var lvl Level
	var file string
	good:=true
	var errstr string
	filt, good,file = defaultFileLogWriter(filename+".sys."+time.Now().Format("2006-01-02")+".log",filepath,islog)
	if !good {
		errstr="can't create log file:"+filepath+filename+".sys."+time.Now().Format("2006-01-02")+".log"
		return errors.New(errstr)
	}
	lvl=INFO
	log[file] = &Filter{lvl, filt}
	filt, good,file = defaultFileLogWriter(filename+".run."+time.Now().Format("2006-01-02")+".log",filepath,islog)
	if !good {
		errstr="can't create log file:"+filepath+filename+".run."+time.Now().Format("2006-01-02")+".log"
		return errors.New(errstr)
	}
	lvl=FINEST
	log[file] = &Filter{lvl, filt}
	filt, good,file = defaultFileLogWriter(filename+".err."+time.Now().Format("2006-01-02")+".log",filepath,islog)
	if !good {
		errstr="can't create log file:"+filepath+filename+".err."+time.Now().Format("2006-01-02")+".log"
		return errors.New(errstr)
	}
	lvl=ERROR
	log[file] = &Filter{lvl, filt}
	return nil
}



func defaultFileLogWriter(filename string, filepath string,islog bool) (*FileLogWriter, bool, string) {
	file := filepath+filename
	format := "[%D %T] [%L] (%S) %M"
	maxlines := 100000000
	maxsize := 10000000000
	daily := true
	rotate := false

	if len(file) == 0 {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n", "filename", filename)
		return nil, false, file
	}

	flw := NewFileLogWriter(file, rotate,islog)
	if flw == nil{
		return nil,false,file
	}
	fmt.Println("file:[%s]",file)
	flw.SetFormat(format)
	flw.SetRotateLines(maxlines)
	flw.SetRotateSize(maxsize)
	flw.SetRotateDaily(daily)
	return flw, true, file
}




func CreateDir(str []string){
	for _,v:=range str{
		exist, err := PathExists(v)
		if err != nil {
			fmt.Printf("get dir error![%v]\n", err)
			return
		}
		if exist {
			fmt.Printf("has dir![%v]\n", v)
		} else {
			err := os.Mkdir(v, os.ModePerm)
			if err != nil {
				fmt.Printf("mkdir failed![%v]\n", err)
			}
		}
	}
}


func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func splitpath(tmpstr string,str * []string){
	num:=-1
	start:=strings.Index(tmpstr,"/")
	startData:= string([]byte(tmpstr)[:start+1])
	tmpData:= string([]byte(tmpstr)[:])
	fmt.Printf("%s\n",startData)
	for true {
		startData= string([]byte(tmpData)[:start])
		tmpData=string([]byte(tmpData)[start+1:])
		if(num==-1) {
			*str = append(*str, startData)
			num++
		}else{
			*str = append(*str,(*str)[num]+"/"+startData)
			num++
		}
		//parse
		//writedb
		start=strings.Index(tmpData,"/")
		if(start == -1){
			startData=tmpData
			if(num==-1) {
				*str = append(*str, startData)
				num++
			}else{
				*str = append(*str,(*str)[num]+"/"+startData)
				num++
			}
			//parse
			//writedb
			break
		}
	}
}







