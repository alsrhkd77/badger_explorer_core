# Badger Explorer Core - Subprocess API Specification

Badger Explorer Core는 다른 애플리케이션(예: Electron, Python 스크립트 등)에서 하위 프로세스로 실행되어 BadgerDB와 상호작용할 수 있도록 JSON-Lines 기반의 RPC API를 제공합니다.

## 실행 방법

```bash
badger_explorer_core -mode subprocess
```

실행 후 표준 입력(Stdin)으로 요청을 보내고, 표준 출력(Stdout)으로 응답을 받습니다. 각 메시지는 개행 문자(`\n`)로 구분된 JSON 객체여야 합니다.

## 공통 데이터 구조

### 요청 (Request)

```json
{
  "id": "unique_request_id",
  "type": "request_type",
  "params": { ... }
}
```

### 응답 (Response)

성공 시:
```json
{
  "id": "unique_request_id",
  "type": "request_type_resp",
  "result": { ... }
}
```

실패 시:
```json
{
  "id": "unique_request_id",
  "type": "error",
  "error": {
    "code": 1000,
    "message": "Error description"
  }
}
```

---

## API 목록

### 1. DB 열기 (`open_db`)

지정된 경로의 BadgerDB를 엽니다. Windows 호환성을 위해 항상 Read-Write 모드로 열립니다.

**Params:**
- `path` (string): DB 디렉토리 절대 경로

**Result:** `null`

**Example:**
```json
{"id":"1", "type":"open_db", "params":{"path":"C:\\Data\\badger"}}
```

### 2. 키 목록 조회 (`list_keys`)

조건에 맞는 키 목록을 조회합니다.

**Params:**
- `prefix` (string): 검색어 (접두사, 부분문자열, 또는 정규식)
- `mode` (string): 검색 모드 (`"prefix"`, `"substring"`, `"regex"`)
- `sort` (string): 정렬 순서 (`"asc"`, `"desc"`)
- `limit` (int): 조회할 최대 항목 수
- `offset` (int): 건너뛸 항목 수

**Result:**
- `keys` (Array): 키 항목 리스트
  - `Key` (string): 키
  - `ValuePreview` (string): 값 미리보기
  - `Size` (int64): 값 크기 (bytes)
  - `ExpiresAt` (uint64): 만료 타임스탬프
- `has_more` (bool): 더 많은 항목이 있는지 여부

**Example:**
```json
{"id":"2", "type":"list_keys", "params":{"prefix":"user:", "mode":"prefix", "limit":20, "offset":0}}
```

### 3. 값 조회 (`get_value`)

특정 키의 전체 값을 조회합니다.

**Params:**
- `key` (string): 조회할 키

**Result:**
- `value` (string): Base64 인코딩된 값

**Example:**
```json
{"id":"3", "type":"get_value", "params":{"key":"user:123"}}
```

### 4. 값 쓰기 (Chunked Upload)

큰 값을 효율적으로 전송하기 위해 3단계 프로세스(`put_value` -> `put_chunk` -> `put_commit`)를 사용합니다.

#### 4-1. 쓰기 초기화 (`put_value`)

**Params:**
- `key` (string): 저장할 키
- `value_length` (int): 전체 값의 크기 (bytes)
- `ttl` (int): TTL (초 단위, 0이면 무제한)

**Result:** `null`

#### 4-2. 청크 전송 (`put_chunk`)

**Params:**
- `id` (string): `put_value` 요청의 ID (세션 식별용)
- `chunk_index` (int): 청크 인덱스 (0부터 시작)
- `data` (string): Base64 인코딩된 청크 데이터

**Result:** `null`

#### 4-3. 쓰기 확정 (`put_commit`)

**Params:**
- `id` (string): `put_value` 요청의 ID
- `key` (string): 저장할 키
- `ttl` (int): TTL

**Result:** `null`

### 5. 키 삭제 (`delete_key`)

특정 키를 삭제합니다.

**Params:**
- `key` (string): 삭제할 키

**Result:** `null`

**Example:**
```json
{"id":"6", "type":"delete_key", "params":{"key":"user:123"}}
```

### 6. DB 닫기 (`close_db`)

현재 열려있는 DB를 닫습니다.

**Params:** 없음

**Result:** `null`
