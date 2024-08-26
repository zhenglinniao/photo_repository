package updata

import (
	"fmt"
	"photoAlbum/config"
	"photoAlbum/global"
	"photoAlbum/service"
	"time"
)

func StartPhotoAlbumUpdater() (stop func()) {
	ticker := time.NewTicker(1 * time.Minute)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println("Tick at", t)

				var err error
				global.Config, err = config.New("./config.yaml")
				if err != nil {
					fmt.Println("Error loading config:", err)
					continue // 继续下一次循环，而不是返回
				}

				global.PhotoAlbumList, err = service.InitPhotoAlbum(global.Config.PhotoAlbumAbsolutePath)
				if err != nil {
					fmt.Println("Error initializing photo album:", err)
					continue // 继续下一次循环，而不是返回
				}
			}
		}
	}()

	return func() {
		ticker.Stop()
		done <- true
	}
}
