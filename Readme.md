# epub文件转换为gitbook

### 使用方式

```
docker run -d -e bookname="目标文件名" -p 4000:4000 -v /目标文件所在地址:/gitbook epub_to_gitbook
```