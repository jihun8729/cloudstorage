#!/bin/bash

# 업로드할 파일 경로
FILE="./test.tif"

# 업로드 대상 리모트들
REMOTES=("gdrive1" "gdrive2" "one-drive1" "one-drive2")

# 파일 크기 (bytes)
FILE_SIZE=$(stat -c %s "$FILE")

# 전체 시작 시간
START_TIME=$(date +%s)

# 각 리모트에 파일 업로드
for REMOTE in "${REMOTES[@]}"; do
    echo "${REMOTE}에 업로드 시작"
    ./rclone copy "$FILE" "${REMOTE}:/"
    if [ $? -eq 0 ]; then
        echo " ${REMOTE}에 업로드 성공"
    else
        echo " ${REMOTE}에 업로드 실패"
    fi
done

# 전체 종료 시간
END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))


# 출력
echo "총 업로드 시간: ${ELAPSED}초"

read -p "Enter를 누르면 창이 닫힙니다..."
