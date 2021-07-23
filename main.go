package main

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	chapTemp     = `* [%s](%s.md)`
	summaryStart = `# Summary

* [简介](README.md)
`
	OEBPS       = "OEBPS/"
	TextPath    = "OEBPS/Text"
	IncludeTemp = `!INCLUDE "%s"`
)

var (
	mimeTypePath = "mimetype"
	contentPath  = OEBPS + "content.opf"
	tocPath      = OEBPS + "toc.ncx"
	output       = "./output"
	HTMLPath     = output + "/HTML"
	bookPath     = "/gitbook/"
	bookName     = "book1.epub"
)

func init() {
	bookName = os.Getenv("bookname")
}
func main() {
	zf, err := zip.OpenReader(bookPath + bookName)
	if err != nil {
		panic(err)
	}
	defer zf.Close()
	// for _, v := range zf.File {
	// 	fmt.Println(v.Name)
	// }

	mimeType := findFileByName(zf.File, mimeTypePath)
	if mimeType == nil {
		panic("there is not mimeType")
	}
	mimeTypeStr, err := getFileContent(mimeType)
	if err != nil {
		panic(err)
	}
	if mimeTypeStr != "application/epub+zip" {
		panic("mimeType is not epub")
	}

	catalogInfo, err := getcatalog(findFileByName(zf.File, tocPath))
	if err != nil {
		panic(err)
	}

	sort.Slice(catalogInfo, func(i, j int) bool {
		return catalogInfo[i].Order < catalogInfo[j].Order
	})
	// for _, v := range catalogInfo {
	// 	fmt.Printf("%+v\n", v.Path)
	// }
	_, err = os.Stat(output)
	if err != nil {
		os.Mkdir(output, os.ModePerm)
	}
	err = makeSummary(catalogInfo)
	if err != nil {
		panic(err)
	}
	contentToc, err := getContentToc(findFileByName(zf.File, contentPath))
	if err != nil {
		panic(err)
	}
	// fmt.Println(contentToc)
	err = makeReadMe(findFileByName(zf.File, contentPath))
	if err != nil {
		panic(err)
	}

	err = makeContentFile(zf.File, catalogInfo, &contentToc)
	if err != nil {
		panic(err)
	}
	err = UnzipOEBPSinfo(zf.File)
	if err != nil {
		panic(err)
	}
}

func findFileByName(files []*zip.File, fileName string) *zip.File {
	for _, v := range files {
		if strings.HasSuffix(v.Name, fileName) {
			return v
		}
	}
	return nil
}

func getFileContent(file *zip.File) (string, error) {
	if file == nil {
		return "", fmt.Errorf("nil file")
	}
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	con, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(con), nil
}

func getContentToc(file *zip.File) (ans []string, err error) {
	if file.Name != contentPath {
		return ans, fmt.Errorf("content.opt miss")
	}
	f, err := file.Open()
	if err != nil {
		return
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	doc.Find("spine").Find("itemref").Each(func(i int, selection *goquery.Selection) {
		ref, exist := selection.Attr("idref")
		if exist {
			ans = append(ans, ref)
		}
	})
	if len(ans) == 0 {
		return ans, fmt.Errorf("can not read itemRef")
	}
	return
}

type Catalog struct {
	Content string
	Order   int
	Path    string
}

func getcatalog(file *zip.File) (catalogInfo []Catalog, err error) {
	if file.Name != tocPath {
		return nil, fmt.Errorf("catalog file miss")
	}
	f, err := file.Open()
	if err != nil {
		return
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return
	}
	depth := true
	doc.Find("meta").Each(func(i int, selection *goquery.Selection) {
		if val, exist := selection.Attr("name"); exist && val == "dtb:depth" {
			if val, _ := selection.Attr("content"); val != "1" {
				depth = false

			}
		}
	})
	if !depth {
		return catalogInfo, fmt.Errorf("can not make catalog which depth more than 1")
	}

	// title := doc.Find("docTitle")
	// fmt.Println(strings.TrimSpace(title.Text()))

	doc.Find("navMap").Find("navPoint").EachWithBreak(func(i int, navPoint *goquery.Selection) bool {
		tmp := Catalog{}
		tmp.Content = strings.TrimSpace(navPoint.Text())
		order, exist := navPoint.Attr("playorder")

		if !exist {
			err = fmt.Errorf("require playOrder in toc.ncx")
			return false
		}
		tmp.Order, err = strconv.Atoi(order)
		if err != nil {
			return false
		}
		tmp.Path, exist = navPoint.Find("content").Attr("src")
		if !exist {
			err = fmt.Errorf("require content src in toc.ncx")
			return false
		}
		catalogInfo = append(catalogInfo, tmp)
		return true
	})
	if err != nil {
		return
	}
	return

}

func makeSummary(catalogInfo []Catalog) error {
	summaryStr := summaryStart
	for _, v := range catalogInfo {
		summaryStr += fmt.Sprintf(chapTemp, v.Content, v.Content) + "\n"
	}
	return makeFile(output+"/SUMMARY.md", summaryStr)
}

func makeFile(topic, content string) error {
	f, err := os.Create(topic)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}

func makeReadMe(file *zip.File) error {
	if file.Name != contentPath {
		return fmt.Errorf("content.opt miss")
	}
	f, err := file.Open()
	if err != nil {
		return err
	}
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return err
	}
	return makeFile(output+"/README.md", doc.Find("metadata").Text())
}

func makeContentFile(files []*zip.File, catalogInfo []Catalog, contentToc *[]string) error {
	_, err := os.Stat(HTMLPath)
	if err != nil {
		os.Mkdir(HTMLPath, os.ModePerm)
	}
	for i := 0; i < len(catalogInfo); i++ {
		now := catalogInfo[i]
		err := makeFile(output+"/"+now.Content+".md", fmt.Sprintf(IncludeTemp, "./HTML/"+path.Base(now.Path)))
		if err != nil {
			return err
		}
		f, err := os.Create(HTMLPath + "/" + path.Base(now.Path))
		if err != nil {
			return err
		}
		defer f.Close()

		for len(*contentToc) != 0 {
			nowHtmlFileName := (*contentToc)[0]
			if i != len(catalogInfo)-1 && path.Base(catalogInfo[i+1].Path) == nowHtmlFileName {
				break
			}
			nowHtmlFile := findFileByName(files, nowHtmlFileName)
			if nowHtmlFile == nil {
				return fmt.Errorf("can not find %s", nowHtmlFileName)
			}
			htmlFile, err := nowHtmlFile.Open()
			if err != nil {
				return err
			}
			htmlContent, err := ioutil.ReadAll(htmlFile)
			if err != nil {
				return err
			}

			_, err = f.WriteString(strings.ReplaceAll(string(htmlContent), "../Images", "./Images"))
			if err != nil {
				return err
			}
			(*contentToc) = (*contentToc)[1:]
		}
	}

	return nil
}

func UnzipOEBPSinfo(files []*zip.File) error {
	for _, v := range files {
		name := v.Name
		if strings.HasPrefix(name, OEBPS) && !strings.HasPrefix(name, TextPath) &&
			!strings.HasPrefix(name, contentPath) && !strings.HasPrefix(name, tocPath) {
			last := strings.Replace(v.Name, OEBPS, "", 1)
			dir := path.Dir(last)
			_, err := os.Stat(dir)
			if err != nil {
				os.MkdirAll(output+"/"+dir, os.ModePerm)
			}
			newFile, err := os.Create(output + "/" + last)
			if err != nil {
				return err
			}
			defer newFile.Close()
			zfile, err := v.Open()
			if err != nil {
				return err
			}
			defer zfile.Close()
			zfileCon, err := ioutil.ReadAll(zfile)
			if err != nil {
				return err
			}
			_, err = newFile.Write(zfileCon)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
