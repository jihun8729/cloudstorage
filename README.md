
[<img src="https://rclone.org/img/logo_on_light__horizontal_color.svg" width="50%" alt="rclone logo">](https://rclone.org/#gh-light-mode-only)
[<img src="https://rclone.org/img/logo_on_dark__horizontal_color.svg" width="50%" alt="rclone logo">](https://rclone.org/#gh-dark-mode-only)

[Website](https://rclone.org) |
[Documentation](https://rclone.org/docs/) |
[Download](https://rclone.org/downloads/) |
[Contributing](CONTRIBUTING.md) |
[Changelog](https://rclone.org/changelog/) |
[Installation](https://rclone.org/install/) |
[Forum](https://forum.rclone.org/)

[![Build Status](https://github.com/rclone/rclone/workflows/build/badge.svg)](https://github.com/rclone/rclone/actions?query=workflow%3Abuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/rclone/rclone)](https://goreportcard.com/report/github.com/rclone/rclone)
[![GoDoc](https://godoc.org/github.com/rclone/rclone?status.svg)](https://godoc.org/github.com/rclone/rclone)
[![Docker Pulls](https://img.shields.io/docker/pulls/rclone/rclone)](https://hub.docker.com/r/rclone/rclone)

# Rclone

다양한 Public Cloud Storage 서비스들을 연동시켜서 파일 업로드를 가능하게 하는 오픈소스
<br> <br>
Rclone에 대한 자세한 정보는 위 링크 참조 부탁드립니다.

## Storage providers

Rclone 이 지원하는 Public Cloud Storage 목록들 [the full list of all storage providers and their features](https://rclone.org/overview/)

# 기초 환경 구축 및 설치 방법
## 1. Cloud Aggregator 설치
```
git clone https://github.com/HandongSF/cloud_storage.git
```
## 2. Go version 1.21 이상 설치
설치
```
brew install go
```
설치 확인
```
go version
```

## 3. Rclone build
클론 완료한 폴더 이동 후 rclone build 
```
go build -o rclone
```
빌드 확인
```
./rclone version
```
전역에서 쓸 수 있게 변수 설정 (MAC 기준)
```
sudo cp ./rclone /usr/local/bin/
```
```
sudo chmod +x /usr/local/bin/rclone
```
### 전역에서 사용이 불가능한 경우
임시적으로 rclone 실행 파일 생성 후 ./rclone ~~ 로 실행하시면 됩니다.
```
go build -o rclone
```
## 4. Public Storage 연결
rclone config 명령어 실행 후 안내되는 사항에 따라서 진행하시면 됩니다.
```
rclone config
```

# 각 명령어 실행 방법
## 1. dis_upload (분산 업로드)
```
rclone dis_upload [파일경로] -b [알고리즘 이름]
알고리즘 이름 : RoundRobin, ResourceBased, DownloadOptima, UploadOptima 중 택 1(대소문자 상관 있습니다)
```
## 2. dis_download (분산 다운로드)
```
rclone dis_download [파일 이름] [다운 받을 로컬 위치 경로]
```
## 3. dis_ls (분산 업로드 된 파일 조회)
```
rclone dis_ls
```
## 4. dis_rm (분산 업로드 된 파일 삭제)
```
rclone dis_rm [파일 이름]
```
## 5. dis_config (몰?루?)
```
?? 몰?루?
```
