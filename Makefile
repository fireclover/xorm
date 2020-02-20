IMPORT := xorm.io/xorm
export GO111MODULE=on

GO ?= go
GOFMT ?= gofmt -s
TAGS ?=
SED_INPLACE := sed -i

GOFILES := $(shell find . -name "*.go" -type f)

PACKAGES ?= $(shell GO111MODULE=on $(GO) list ./...)

TEST_MYSQL_HOST ?= mysql:3306
TEST_MYSQL_DBNAME ?= xorm_test
TEST_MYSQL_USERNAME ?= root
TEST_MYSQL_PASSWORD ?=
TEST_PGSQL_HOST ?= pgsql:5432
TEST_PGSQL_DBNAME ?= testgitea
TEST_PGSQL_USERNAME ?= postgres
TEST_PGSQL_PASSWORD ?= postgres
TEST_MSSQL_HOST ?= mssql:1433
TEST_MSSQL_DBNAME ?= gitea
TEST_MSSQL_USERNAME ?= sa
TEST_MSSQL_PASSWORD ?= MwantsaSecurePassword1

.PHONY: all
all: build

.PHONY: build
build: go-check $(GO_SOURCES)
	$(GO) build

.PHONY: clean
clean:
	$(GO) clean -i ./...
	rm -rf *.sql *.log test.db

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GOFILES)

.PHONY: fmt-check
fmt-check:
	# get all go files and run go fmt on them
	@diff=$$($(GOFMT) -d $(GOFILES)); \
	if [ -n "$$diff" ]; then \
		echo "Please run 'make fmt' and commit the result:"; \
		echo "$${diff}"; \
		exit 1; \
	fi;

.PHONY: go-check
go-check:
	$(eval GO_VERSION := $(shell printf "%03d%03d%03d" $(shell go version | grep -Eo '[0-9]+\.?[0-9]+?\.?[0-9]?\s' | tr '.' ' ');))
	@if [ "$(GO_VERSION)" -lt "001011000" ]; then \
		echo "Gitea requires Go 1.11.0 or greater to build. You can get it at https://golang.org/dl/"; \
		exit 1; \
	fi

.PHONY: help
help:
	@echo "Make Routines:"
	@echo " -                   equivalent to \"build\""
	@echo " - build             creates the entire project"
	@echo " - clean             delete integration files and build files but not css and js files"
	@echo " - fmt               format the code"
	@echo " - lint            	run code linter revive"
	@echo " - misspell          check if a word is written wrong"
	@echo " - test       		run default unit test"
	@echo " - test-sqlite       run unit test for sqlite"
	@echo " - vet               examines Go source code and reports suspicious constructs"

.PHONY: lint
lint: revive

.PHONY: revive
revive:
	@hash revive > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/mgechev/revive; \
	fi
	revive -config .revive.toml -exclude=./vendor/... ./... || exit 1

.PHONY: misspell
misspell:
	@hash misspell > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/client9/misspell/cmd/misspell; \
	fi
	misspell -w -i unknwon $(GOFILES)

.PHONY: misspell-check
misspell-check:
	@hash misspell > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/client9/misspell/cmd/misspell; \
	fi
	misspell -error -i unknwon,destory $(GOFILES)

.PHONY: test
test: test-sqlite

.PNONY: test-mssql
test-mssql: go-check
	$(GO) test -race -db=mssql -conn_str="server=$(TEST_MYSQL_HOST);user id=$(TEST_MYSQL_USERNAME);password=$(TEST_MYSQL_PASSWORD);database=$(TEST_MYSQL_DBNAME)"

.PNONY: test-mssql-cache
test-mssql-cache: go-check
	$(GO) test -race -db=mssql -cache=true -conn_str="server=$(TEST_MYSQL_HOST);user id=$(TEST_MYSQL_USERNAME);password=$(TEST_MYSQL_PASSWORD);database=$(TEST_MYSQL_DBNAME)"


.PNONY: test-mymysql
test-mymysql: go-check
	$(GO) test -race -db=mymysql -conn_str="tcp:$(TEST_MYSQL_HOST)*$(TEST_MYSQL_DBNAME)/$(TEST_MYSQL_USERNAME)/$(TEST_MYSQL_PASSWORD)"

.PNONY: test-mymysql-cache
test-mymysql-cache: go-check
	$(GO) test -race -db=mymysql -cache=true -conn_str="tcp:$(TEST_MYSQL_HOST)*$(TEST_MYSQL_DBNAME)/$(TEST_MYSQL_USERNAME)/$(TEST_MYSQL_PASSWORD)"

.PNONY: test-mysql
test-mysql: go-check
	$(GO) test -race -db=mysql -conn_str="$(TEST_MYSQL_USERNAME):$(TEST_MYSQL_PASSWORD)@tcp($(TEST_MYSQL_HOST))/$(TEST_MYSQL_DBNAME)"

.PNONY: test-mysql-cache
test-mysql-cache: go-check
	$(GO) test -race -db=mysql -cache=true -conn_str="$(TEST_MYSQL_USERNAME):$(TEST_MYSQL_PASSWORD)@tcp($(TEST_MYSQL_HOST))/$(TEST_MYSQL_DBNAME)"

.PHONY: test-mysql\#%
test-mysql\#%: go-check
	$(GO) test -race -run $* -db=mysql -conn_str="$(TEST_MYSQL_USERNAME):$(TEST_MYSQL_PASSWORD)@tcp($(TEST_MYSQL_HOST))/$(TEST_MYSQL_DBNAME)"

.PNONY: test-postgres
test-postgres: go-check
	$(GO) test -race -db=postgres -conn_str="postgres://$(TEST_MYSQL_USERNAME):$(TEST_MYSQL_PASSWORD)@$(TEST_MYSQL_HOST)/$(TEST_MYSQL_DBNAME)?sslmode=disable"

.PNONY: test-postgres-cache
test-postgres-cache: go-check
	$(GO) test -race -db=postgres -cache=true -conn_str="postgres://$(TEST_MYSQL_USERNAME):$(TEST_MYSQL_PASSWORD)@$(TEST_MYSQL_HOST)/$(TEST_MYSQL_DBNAME)?sslmode=disable"

.PHONY: test-postgres\#%
test-postgres\#%: go-check
	$(GO) test -race -run $* -db=postgres -conn_str="postgres://$(TEST_MYSQL_USERNAME):$(TEST_MYSQL_PASSWORD)@$(TEST_MYSQL_HOST)/$(TEST_MYSQL_DBNAME)?sslmode=disable"

.PHONY: test-sqlite
test-sqlite: go-check
	$(GO) test -race -db=sqlite3 -conn_str="./test.db?cache=shared&mode=rwc"

.PHONY: test-sqlite-cache
test-sqlite-cache: go-check
	$(GO) test -race -cache=true -db=sqlite3 -conn_str="./test.db?cache=shared&mode=rwc"

.PHONY: test-sqlite\#%
test-sqlite\#%: go-check
	$(GO) test -race -run $* -db=sqlite3 -conn_str="./test.db?cache=shared&mode=rwc"

.PNONY: test-tidb
test-tidb: go-check
	$(GO) test -race -db=mysql -ignore_select_update=true -conn_str="$(TEST_MYSQL_USERNAME):$(TEST_MYSQL_PASSWORD)@tcp($(TEST_MYSQL_HOST))/$(TEST_MYSQL_DBNAME)"

.PNONY: test-tidb-cache
test-tidb-cache: go-check
	$(GO) test -race -db=mysql -ignore_select_update=true -cache=true -conn_str="$(TEST_MYSQL_USERNAME):$(TEST_MYSQL_PASSWORD)@tcp($(TEST_MYSQL_HOST))/$(TEST_MYSQL_DBNAME)"

.PHONY: test-tidb\#%
test-tidb\#%: go-check
	$(GO) test -race -run $* -db=mysql -ignore_select_update=true -conn_str="$(TEST_MYSQL_USERNAME):$(TEST_MYSQL_PASSWORD)@tcp($(TEST_MYSQL_HOST))/$(TEST_MYSQL_DBNAME)"


go test -db=mysql -conn_str="root:@tcp(localhost:4000)/xorm_test" -ignore_select_update=true
.PHONY: vet
vet:
	$(GO) vet $(PACKAGES)