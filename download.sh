#!/bin/bash

# 리모트들 (순차적으로 시도)
REMOTES=("gdrive1" "gdrive2" "one-drive" "dropbox")

# 다운로드할 파일 경로 (리모트 내부 경로)
REMOTE_PATH="test.tif"

# 저장할 로컬 경로 (현재 디렉토리)
LOCAL_PATH="./$REMOTE_PATH"

# 기존 파일 삭제 (테스트 중복 방지용, 필요시 주석처리)
[ -f "$LOCAL_PATH" ] && rm -f "$LOCAL_PATH"

# 전체 시작 시간
START_TIME=$(date +%s)

# 다운로드 성공 여부
SUCCESS=0

for REMOTE in "${REMOTES[@]}"; do
    echo "[$(date)] ${REMOTE}에서 다운로드 시도 중..."
    
    ./rclone copy "${REMOTE}:/$REMOTE_PATH" ./
    if [ -f "$LOCAL_PATH" ]; then
        echo "[$(date)] ${REMOTE}에서 다운로드 성공"
        SUCCESS=1
        break
    else
        echo "[$(date)] ${REMOTE}에서 다운로드 실패"
    fi
done

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))



    echo "총 다운로드 시간: ${ELAPSED}초"
    read -p "Enter를 누르면 창이 닫힙니다..."
else
    echo "모든 리모트에서 다운로드 실패"
fi


