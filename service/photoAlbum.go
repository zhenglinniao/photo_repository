package service

import (
	"fmt"
	"image"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"photoAlbum/global"
	"photoAlbum/models"
	"photoAlbum/pkg/utils"
	"sort"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2/log"
	"gopkg.in/yaml.v3"
)

func InitPhotoAlbum(root string) (*models.PhotoAlbums, error) {

	photoAlbums, err := readPhotoAlbum(root)

	if err != nil {
		return nil, err
	}
	sort.Sort(photoAlbums)

	return &photoAlbums, nil
}

func readPhotoAlbum(absolutePath string) (models.PhotoAlbums, error) {

	// 创建一个通道，用于存储相册集
	resultCh := make(chan *models.PhotoAlbum, 10)

	// 创建一个等待组，用于等待所有协程完成
	wg := sync.WaitGroup{}

	// 遍历指定路径下的所有文件
	err := filepath.Walk(absolutePath, func(path string, info os.FileInfo, err error) error {
		// 如果发生错误，返回错误
		if err != nil {
			return err
		}
		// 如果不是Yaml文件，跳过
		if !utils.IsYamlFile(info) {
			return nil
		}
		//因为有Yaml文件,当前被认为是一个相册集
		//如果同一个文件夹有多个Yaml，也会被当多个相册集
		// 增加等待组的计数
		wg.Add(1)
		// 启动一个协程，解析相册集
		go func() {
			// 协程完成时，减少等待组的计数
			defer wg.Done()
			// 解析相册集
			log.Info("path", path)
			log.Info("absolutePath", absolutePath)
			pa := parserPhotoAlbum(absolutePath, path)
			// 将解析结果发送到通道
			resultCh <- &pa
		}()
		// 继续遍历下一个文件
		return nil
	})

	// 如果遍历过程中发生错误，返回错误
	if err != nil {
		return nil, err
	}
	// 启动一个协程，等待所有协程完成，并关闭通道
	go func() {
		// 等待所有协程完成
		wg.Wait()
		// 关闭通道
		close(resultCh)
	}()

	// 创建一个空的相册集
	var photoAlbums models.PhotoAlbums
	// 从通道中读取所有相册集，并添加到相册集中
	for pa := range resultCh {
		photoAlbums = append(photoAlbums, *pa)
	}

	// 返回相册集
	return photoAlbums, nil
}

func parserPhotoAlbum(root, path string) models.PhotoAlbum {
	// 解析yaml文件
	pa, err := parserYaml(path)
	if err != nil {
		// 如果解析失败，将错误信息赋值给pa的Error字段，并返回pa
		pa.Error = err
		return pa
	}
	// 获取path所在的目录
	dir := filepath.Dir(path)
	// 获取path相对于root的相对路径
	relPath, err := filepath.Rel(root, dir)
	if err != nil {
		// 如果获取相对路径失败，将错误信息赋值给pa的Error字段，并返回pa
		pa.Error = err
		return pa
	}
	// 解析照片
	pa.Photos, err = parserPhotos(dir)
	if err != nil {
		// 如果解析照片失败，将错误信息赋值给pa的Error字段，并返回pa
		pa.Error = err
		return pa
	}
	// 将相对路径赋值给pa的Path字段
	pa.Path = models.PhotoAlbumPath(relPath)
	// 将照片数量赋值给pa的Count字段
	pa.Count = len(pa.Photos)

	// 返回pa
	return pa
}

func parserPhotos(dir string) (models.Photos, error) {
	// 定义一个Photos类型的变量
	var photos models.Photos
	log.Info("dir:", dir)
	// 读取指定目录下的文件
	files, err := ioutil.ReadDir(dir)
	// 如果读取失败，返回空的照片列表和错误信息
	if err != nil {
		return photos, err
	}
	// 遍历文件列表
	for _, file := range files {
		// 如果文件不是目录
		if !file.IsDir() {
			// 获取文件后缀名
			ext := strings.ToLower(filepath.Ext(file.Name()))
			// 如果后缀名是.jpg或.jpeg
			if ext == ".jpg" || ext == ".jpeg" {
				// 解析照片数据
				log.Info("dir:", dir)
				log.Info("file:", file)
				photo := parsePhotoData(dir, file)
				// 将照片添加到照片列表中
				photos = append(photos, photo)
			}
		}
	}

	// 返回照片列表和nil
	return photos, nil
}

func parserYaml(path string) (models.PhotoAlbum, error) {
	// 定义一个PhotoAlbum类型的变量
	var photoAlbum models.PhotoAlbum
	// 读取指定路径的文件内容
	content, err := ioutil.ReadFile(path)
	// 如果读取文件出错，则返回空PhotoAlbum和错误信息
	if err != nil {
		return photoAlbum, err
	}
	log.Info("content:", string(content))
	// 将读取到的文件内容解析为PhotoAlbum类型
	err = yaml.Unmarshal(content, &photoAlbum)
	// 如果解析出错，则返回空PhotoAlbum和错误信息
	if err != nil {
		return photoAlbum, err
	}
	// 返回解析后的PhotoAlbum和nil
	return photoAlbum, nil
}

// 解析图片元数据
func parsePhotoData(dir string, file fs.FileInfo) models.Photo {

	// 创建一个空的Photo对象
	photo := models.Photo{}

	// 获取文件路径
	filePath := filepath.Join(dir, file.Name())
	// 构建封面路径
	coverPath, err := buildCoverPath(filePath)
	// 如果构建封面路径出错，则将错误信息赋值给photo对象的Error字段，并返回photo对象
	if err != nil {
		photo.Error = err
		return photo
	}
	// 打开文件
	img, err := imaging.Open(filePath)
	// 如果打开文件出错，则将错误信息赋值给photo对象的Error字段，并返回photo对象
	if err != nil {
		photo.Error = err
		return photo
	}

	// 获取图片的宽度和高度
	photo.Width = img.Bounds().Dx()
	photo.Height = img.Bounds().Dy()
	// 获取图片的大小，单位为MB
	photo.Size = fmt.Sprintf("%.2f", float64(file.Size())/(1024*1024))
	// 获取图片的格式
	photo.Format = filepath.Ext(file.Name())
	// 获取图片的名称，不包括格式
	photo.Name = strings.TrimSuffix(file.Name(), photo.Format)

	// 根据路径解析图片的EXIF信息
	photo.ParseExifByPath(filePath)

	// 构建图片的封面
	err = buildPhotoCover(img, coverPath)
	// 如果构建封面出错，则将错误信息赋值给photo对象的Error字段，并返回photo对象
	if err != nil {
		photo.Error = err
		return photo
	}

	// 返回photo对象
	return photo
}

// buildPhotoCover 函数用于生成图片封面
func buildPhotoCover(img image.Image, coverPath string) error {
	// 判断coverPath是否存在
	if utils.IsFile(coverPath) { //有封面了
		// 打开coverPath
		img, err := imaging.Open(coverPath)
		if err != nil {
			return err
		}
		// 判断图片高度是否与封面高度一致
		if img.Bounds().Dy() == global.Config.CoverHeight {
			return nil
		}
	}
	// 获取coverPath的目录
	coverDir := filepath.Dir(coverPath)
	// 判断coverDir是否存在
	if !utils.IsDir(coverDir) {
		// 创建coverDir
		err := utils.MakeDir(coverDir)
		if err != nil {
			return err
		}
	}
	// 调整图片大小为封面高度
	cover := imaging.Resize(img, 0, global.Config.CoverHeight, imaging.NearestNeighbor)
	// 保存图片
	err := imaging.Save(cover, coverPath)
	if err != nil {
		return err
	}
	// 打印coverPath
	fmt.Println(coverPath)
	fmt.Println(coverPath)
	return nil
}

func buildCoverPath(path string) (string, error) {
	// 获取path相对于global.Config.PhotoAlbumAbsolutePath的相对路径
	rel, err := filepath.Rel(global.Config.PhotoAlbumAbsolutePath, path)
	log.Info("rel:", rel)
	// 如果有错误，则返回空字符串和错误
	if err != nil {
		return " ", err
	}
	// 返回path相对于global.Config.PhotoAlbumCoverAbsolutePath的相对路径
	log.Info("global.Config.PhotoAlbumCoverAbsolutePath", global.Config.PhotoAlbumCoverAbsolutePath)
	return filepath.Join(global.Config.PhotoAlbumCoverAbsolutePath, rel), nil
}
