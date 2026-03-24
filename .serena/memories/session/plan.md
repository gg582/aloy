# Plan: aloy - C++ Package Manager in Go

## TL;DR
Go 기반 C++ 패키지 매니저 'aloy'를 구현한다. YAML 설정으로 의존성 관리 + CMake 자동 생성 + 빌드 오케스트레이션을 수행한다.
MVP는 init/add/sync/build 4개 명령, 이후 clean/download/update/remove 순으로 확장.

## Phase 1: Foundation (프로젝트 구조 & 모델 정의)

### Step 1.1 — Go 패키지 구조 스캐폴딩
Go 프로젝트 디렉토리 레이아웃:
```
cmd/
  root.go          - cobra root command
  init.go          - aloy init
  add.go           - aloy add
  sync.go          - aloy sync
  build.go         - aloy build
  clean.go         - aloy clean (2순위)
  download.go      - aloy download (2순위)
  update.go        - aloy update (3순위)
  remove.go        - aloy remove (3순위)
internal/
  model/
    project.go     - project.yaml 구조체 정의
    lockfile.go    - aloy.lock 구조체 정의
  parser/
    yaml.go        - project.yaml 읽기/쓰기
    lock.go        - aloy.lock 읽기/쓰기
  resolver/
    semver.go      - SemVer 범위 파싱 & 교집합 계산
    graph.go       - 의존성 그래프 구축 & 단일 버전 합의
  git/
    clone.go       - git clone/fetch/checkout 래퍼
    tags.go        - 태그 리스트 조회 & SemVer 필터링
  cmake/
    generator.go   - CMakeLists.txt 생성 엔진
    templates.go   - CMake 템플릿 조각들
    scanner.go     - 소스 파일 글로빙 (Go filepath.Glob)
  scaffold/
    init.go        - 프로젝트 초기화 로직
main.go            - cobra Execute() 호출
```

### Step 1.2 — 외부 의존성 도입
- `github.com/spf13/cobra` — CLI 프레임워크
- `gopkg.in/yaml.v3` — YAML 파싱
- `github.com/Masterminds/semver/v3` — SemVer 범위 파싱 & 비교
- 표준 라이브러리: `os/exec` (git/cmake 실행), `path/filepath` (파일 스캔), `crypto/sha256` (락파일 해시)

### Step 1.3 — 데이터 모델 정의 (`internal/model/`)
project.yaml 매핑 구조체:
- `ProjectConfig` — name, version, cxx_standard, targets, inject_cmake
- `Target` — type(executable/library/shared_library/header_only), sources, includes, platforms, dependencies
- `Dependency` — name, git, version, type(normal/system), alias, cmake_options
- `PlatformConfig` — sources, compiler_flags, linker_flags
- `IncludeConfig` — public, private

aloy.lock 구조체:
- `LockFile` — 버전, 패키지 목록
- `LockedPackage` — name, git_url, resolved_version, commit_sha, integrity_hash

## Phase 2: CLI 스켈레톤 & init 명령 (1순위)

### Step 2.1 — cobra CLI 프레임워크 설정
- `cmd/root.go`: rootCmd 정의 (version, help 등)
- `main.go`: `cmd.Execute()` 호출

### Step 2.2 — `aloy init` 구현
- 현재 디렉토리에 기본 `project.yaml` 생성 (이름은 디렉토리명에서 추론)
- `src/`, `include/` 디렉토리 생성
- `.gitignore`에 `.my_modules/`, `build/` 추가 권고 메시지 출력

## Phase 3: add 명령 & YAML 관리 (1순위)

### Step 3.1 — `aloy add <git_url>` 구현
- URL에서 패키지명 추론 (마지막 path segment, .git 제거)
- `--version`, `--alias`, `--system`, `--cmake-option` 플래그 지원
- 기존 `project.yaml` 읽기 → 의존성 추가 → 재작성
- 중복 추가 방지 (이미 존재하면 경고)

## Phase 4: 의존성 해석 & Git 관리 — sync 핵심 (1순위)

### Step 4.1 — Git 래퍼 (`internal/git/`)
- `Clone(url, dest)` — shallow clone으로 `.my_modules/<name>/`에 클론
- `FetchTags(repoPath)` — `git fetch --tags`
- `ListSemVerTags(repoPath)` — 태그 리스트 → SemVer 파싱 → 정렬
- `Checkout(repoPath, ref)` — 특정 태그/SHA로 detach checkout
- 태그 없는 경우: main → master 순으로 폴백, 둘 다 없으면 에러

### Step 4.2 — SemVer 해석기 (`internal/resolver/`)
- `Masterminds/semver` 라이브러리로 범위 파싱 (`^1.2.0`, `~1.2.0`, `>=1.0.0 <2.0.0`)
- `ResolveGraph(rootDeps)` 알고리즘:
  1. 루트 project.yaml 파싱 → 직접 의존성 큐에 추가
  2. BFS로 `.my_modules/<pkg>/project.yaml` 재귀 탐색
  3. 동일 패키지가 여러 버전으로 요구되면 SemVer 교집합 계산
  4. 교집합이 비면 (메이저 충돌) → 에러 + alias 사용 권고 메시지
  5. 교집합 중 최고 버전 선택 → `LockedPackage`로 기록
- 일반 CMake 프로젝트(project.yaml 없는)는 리프 노드로 처리 (재귀 탐색 중단)

### Step 4.3 — 락파일 생성 (`internal/parser/lock.go`)
- 해석 결과를 `aloy.lock`으로 직렬화 (YAML 형식)
- 각 패키지의 commit SHA, resolved version, git URL 기록

## Phase 5: CMake 생성 — sync 완성 (1순위)

### Step 5.1 — 소스 파일 스캐너 (`internal/cmake/scanner.go`)
- `sources` 필드의 glob 패턴 (`src/core/**/*.cpp`) 해석
- Go의 `filepath.Glob` + 재귀 `**` 직접 구현 (doublestar 패턴)
- 결과를 절대 경로가 아닌 프로젝트 루트 상대경로로 반환

### Step 5.2 — CMake 생성기 (`internal/cmake/generator.go`)
마스터 CMakeLists.txt (프로젝트 루트):
- `cmake_minimum_required`, `project()`, `set(CMAKE_CXX_STANDARD)`
- 각 의존성에 대해:
  - aloy 패키지 → `add_subdirectory(.my_modules/<name>)`
  - 일반 CMake 패키지 → `add_subdirectory(.my_modules/<name>)` + cmake_options를 `set()` 으로 주입
  - system 패키지 → `find_package(<Name> REQUIRED)`
- 각 타겟에 대해:
  - `add_executable` / `add_library` / `add_library(INTERFACE)` / `add_library(SHARED)`
  - `target_sources` — 스캔된 파일 목록 하드코딩
  - `target_include_directories` — PUBLIC/PRIVATE 분리
  - `target_link_libraries` — 의존성 타겟 연결
  - 플랫폼별 `if(WIN32)` / `if(UNIX AND NOT APPLE)` 블록
  - `target_compile_options` — 플랫폼별 컴파일러 플래그
- `inject_cmake` 지정 시 `include()` 삽입

의존성 패키지용 CMakeLists.txt (.my_modules/<name>/):
- aloy 패키지인 경우만 생성 (project.yaml 있으면)
- 해당 패키지의 project.yaml을 읽어서 동일 로직으로 생성
- 원본 CMakeLists.txt가 있는 일반 패키지는 건드리지 않음

### Step 5.3 — `aloy sync` 커맨드 통합
1. project.yaml 파싱
2. 의존성 해석 (Step 4.2)
3. Git 클론/체크아웃 (Step 4.1)
4. 락파일 생성/갱신 (Step 4.3)
5. CMake 생성 (Step 5.2)
6. `cmake -B build -S .` 실행 (IDE 인덱싱용)

## Phase 6: build 명령 (1순위)

### Step 6.1 — `aloy build` 구현
- sync가 필요한지 판단 (build/ 없거나, project.yaml이 CMakeLists.txt보다 최신)
- 필요 시 자동으로 sync 호출
- `cmake --build build --config Release` 실행
- `--config` 플래그로 Debug/Release 선택 가능
- 빌드 에러 시 cmake 출력 그대로 전달

## Phase 7: 유틸리티 명령 (2순위)

### Step 7.1 — `aloy clean`
- `build/` 디렉토리 삭제
- `--all` 플래그 시 `.my_modules/`도 삭제
- 루트 CMakeLists.txt는 유지/삭제 선택

### Step 7.2 — `aloy download`
- `aloy.lock` 기반으로 정확한 SHA로 클론만 수행 (CMake 생성 X)
- CI/CD 환경에서 소스 코드 복원 용도

## Phase 8: 관리 명령 (3순위)

### Step 8.1 — `aloy update`
- 각 의존성에 대해 최신 태그 조회
- SemVer 범위 내 최고 버전으로 갱신
- 락파일 업데이트
- `--dry-run` 지원

### Step 8.2 — `aloy remove <name>`
- project.yaml에서 의존성 삭제
- `.my_modules/<name>/` 삭제
- 락파일 재생성

---

## Relevant Files (구현 시 생성/수정)

- `main.go` — cobra Execute() 진입점
- `cmd/root.go` — root 커맨드, 글로벌 플래그
- `cmd/init.go` — init 커맨드
- `cmd/add.go` — add 커맨드
- `cmd/sync.go` — sync 커맨드
- `cmd/build.go` — build 커맨드
- `cmd/clean.go` — clean 커맨드
- `cmd/download.go` — download 커맨드
- `cmd/update.go` — update 커맨드
- `cmd/remove.go` — remove 커맨드
- `internal/model/project.go` — ProjectConfig, Target, Dependency 등
- `internal/model/lockfile.go` — LockFile, LockedPackage
- `internal/parser/yaml.go` — project.yaml 읽기/쓰기
- `internal/parser/lock.go` — aloy.lock 읽기/쓰기
- `internal/resolver/semver.go` — SemVer 범위 파싱 & 교집합
- `internal/resolver/graph.go` — 의존성 그래프 BFS & 단일 버전 합의
- `internal/git/clone.go` — git clone/fetch/checkout 래퍼
- `internal/git/tags.go` — 태그 조회 & SemVer 필터링
- `internal/cmake/generator.go` — CMakeLists.txt 생성 엔진
- `internal/cmake/templates.go` — CMake 코드 조각 템플릿
- `internal/cmake/scanner.go` — glob 기반 소스 파일 스캔
- `internal/scaffold/init.go` — 프로젝트 초기화 로직

## Verification

1. **Unit Tests**: `internal/resolver/` — 다이아몬드 의존성, 메이저 충돌 시나리오 테스트
2. **Unit Tests**: `internal/cmake/scanner.go` — glob 패턴 (`**/*.cpp`) 정확성 테스트
3. **Unit Tests**: `internal/cmake/generator.go` — 생성된 CMakeLists.txt 스냅샷 테스트
4. **Unit Tests**: `internal/git/tags.go` — SemVer 태그 파싱 & 정렬 테스트
5. **Integration Test**: 더미 C++ 프로젝트 + 더미 의존성으로 `init → add → sync → build` 전체 플로우 테스트
6. **수동 검증**: spdlog 같은 실제 오픈소스 패키지를 의존성으로 추가하여 빌드 성공 확인

## Decisions

- Go 1.26.1 사용 (go.mod에 명시)
- CLI: cobra 사용 (Go 생태계 표준)
- SemVer: Masterminds/semver/v3 사용 (범위 연산 내장)
- YAML: gopkg.in/yaml.v3 사용
- 락파일 형식: YAML (project.yaml과 일관성)
- `**` glob: Go 표준 라이브러리에 없으므로 직접 구현 또는 `doublestar` 라이브러리 사용
- header_only 타입 추가 (CMake INTERFACE 라이브러리로 매핑)
- alias 충돌 시 폴더명에 버전 접미사 부여 (예: `logger_v1`, `logger_v2`)
- system 패키지는 find_package() 사용, 향후 매핑 테이블 고도화 가능
- git/cmake는 시스템 PATH에 존재한다고 가정

## Further Considerations

1. **`doublestar` 라이브러리 사용 여부** — `**` glob 지원을 위해 `github.com/bmatcuk/doublestar/v4`를 쓸지, 직접 구현할지. 추천: 라이브러리 사용 (검증됨)
2. **cmake_options 전달 방식** — `set(VAR VALUE CACHE BOOL "")` + `FORCE`로 주입 vs `-D` 플래그로 전달. 추천: `set(... CACHE ... FORCE)` 방식 (add_subdirectory와 호환성 좋음)
3. **에러 처리 전략** — git/cmake 실행 실패 시 stderr 전달 + 종료코드 반환 vs 래핑된 에러 메시지. 추천: 원본 출력 전달 + aloy 컨텍스트 메시지 추가
