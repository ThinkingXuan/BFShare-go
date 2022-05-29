package cmd

import (
	"BFShare-go/internal/encrypt"
	ecc2 "BFShare-go/internal/encrypt/ecc"
	"BFShare-go/internal/encrypt/pre/recrypt"
	"BFShare-go/internal/encrypt/util"
	"BFShare-go/internal/fabric/fabservice"
	file2 "BFShare-go/internal/file"
	"BFShare-go/internal/tools"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"
)

var (
	hash     string
	fileName string
	fileID   string
	dir      string

	startTime2 = time.Now()
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "file download",
	Long:  `file download`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("开始")
		//var fileHash string
		var fileCC fabservice.File
		var fileEncryptEntity fabservice.EncryptEntity

		// get file info
		if fileID != "" {
			fileCC, fileEncryptEntity = QueryFile(fileID)
			//fileHash = fileCC.FileHash
		} else {
			log.Println("file id not is empty")
			return
		}

		// is share's file
		if fileEncryptEntity.NewCapsule != "" && len(fileEncryptEntity.NewCapsule) > 0 {
			//ReCreateKey

			// get myprikey
			myPriKey, _ := util.GetPriKey("config/")
			// get newCapule

			var rekey *ReKey
			var newCapsule *recrypt.Capsule
			//reEncypte { newCapsule}
			_ = json.Unmarshal([]byte(fileEncryptEntity.FileRekey), rekey)
			_ = json.Unmarshal([]byte(fileEncryptEntity.NewCapsule), newCapsule)
			fd, err := recrypt.Decrypt(myPriKey, newCapsule, &rekey.XA, rekey.Fdenc)
			if err != nil {
				log.Println("pre failure")
				return
			}
			// replace
			fileEncryptEntity.FileEncryptCipher = string(fd)
		}

		// get Aes encrypt key
		fileAesKey, err := GetAESKey(fileEncryptEntity.FileEncryptCipher)
		if err != nil {
			log.Println("get aes encrypt key failure,", err)
			return
		}

		fmt.Println("ecc获取解密密钥：", time.Since(startTime2))

		// download file bytes from ipfs according file hash
		fileEncryptBytes, err := file2.DownFile(fileCC.FileHash)
		if err != nil {
			log.Print("error:", err)
			return
		}

		fmt.Println("下载：", time.Since(startTime2))

		// encrypt file
		fileBytes, err := encryptFile(fileEncryptBytes, fileAesKey)
		if err != nil {
			log.Println("file encrypt failure,", err)
		}

		fmt.Println("解密：", time.Since(startTime2))

		// sava file
		dst := filepath.Join(dir, fileCC.FileName)
		err = ioutil.WriteFile(dst, fileBytes, 0666)
		if err != nil {
			log.Print("error:", err)
			return
		}
		fmt.Println("保存文件：", time.Since(startTime2))
		log.Println("file download success")
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringVar(&hash, "hash", "", "file hash")
	downloadCmd.Flags().StringVarP(&fileID, "id", "i", "", "file id")
	downloadCmd.Flags().StringVar(&fileName, "name", "", "file name")
	downloadCmd.Flags().StringVarP(&dir, "dir", "d", "", "download file to dir")
	//_ = downloadCmd.MarkFlagRequired("hash")
}

func QueryFile(fileID string) (fabservice.File, fabservice.EncryptEntity) {
	//service := sdkInit.GetInstance().InitFabric()
	//fileBytes, err := service.QueryFile(fileID)
	//if err != nil {
	//	log.Println("query err:", err)
	//}
	//var file fabservice.File
	//err = json.Unmarshal(fileBytes, &file)
	//if err != nil {
	//	log.Println("Unmarshal err:", err)
	//}
	address := file2.ReadWithFile("config/my.address")
	fileBytes := queryAddressFile(address)
	var filesCC fabservice.AddressFile
	err := json.Unmarshal(fileBytes, &filesCC)
	if err != nil {
		log.Println("Unmarshal err:", err)
	}
	for i := 0; i < len(filesCC.Files); i++ {
		if filesCC.FileEncrypt[i].FileID == fileID {
			return filesCC.Files[i], filesCC.FileEncrypt[i]
		}
	}
	return fabservice.File{}, fabservice.EncryptEntity{}
}
func GetAESKey(fileKeyCipher string) ([]byte, error) {

	ecc := ecc2.NewECC("config/")
	keyPlainText, err := ecc.EccDecrypt(tools.StringToByte(fileKeyCipher))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return keyPlainText, nil
}

func encryptFile(fileBytes []byte, aesKey []byte) ([]byte, error) {
	aes := encrypt.NewAes()
	filePlaintText, err := aes.AESDecrypt(fileBytes, aesKey)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return filePlaintText, nil
}