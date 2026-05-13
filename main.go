package main

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Task構造体: JSONでやり取りするデータの形を定義します
type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// データベースの代わりとなるメモリ上のスライス（配列）
var tasks = []Task{}

// 次に割り当てるタスクID
var nextID = 1

// 1. 【Read】タスク一覧を取得 (GET /tasks)
func getTasks(c echo.Context) error {
	return c.JSON(http.StatusOK, tasks)
}

// 2. 【Create】タスクを新規作成 (POST /tasks)
func createTask(c echo.Context) error {
	task := new(Task)
	// リクエストのJSONを構造体にバインド（当てはめ）する
	if err := c.Bind(task); err != nil {
		return err
	}

	// IDを割り当ててリストに追加
	task.ID = nextID
	nextID++
	tasks = append(tasks, *task)

	return c.JSON(http.StatusCreated, task)
}

// 3. 【Read】指定したIDのタスクを取得 (GET /tasks/:id)
func getTask(c echo.Context) error {
	// URLのパスパラメータ（:id）を文字列から数値に変換
	id, _ := strconv.Atoi(c.Param("id"))

	for _, t := range tasks {
		if t.ID == id {
			return c.JSON(http.StatusOK, t)
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{"message": "タスクが見つかりません"})
}

// 4. 【Update】指定したIDのタスクを更新 (PUT /tasks/:id)
func updateTask(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	updatedTask := new(Task)
	if err := c.Bind(updatedTask); err != nil {
		return err
	}

	for i, t := range tasks {
		if t.ID == id {
			tasks[i].Title = updatedTask.Title
			tasks[i].Completed = updatedTask.Completed
			return c.JSON(http.StatusOK, tasks[i])
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{"message": "タスクが見つかりません"})
}

// 5. 【Delete】指定したIDのタスクを削除 (DELETE /tasks/:id)
func deleteTask(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	for i, t := range tasks {
		if t.ID == id {
			// スライスから該当要素を削除するGoのテクニック
			tasks = append(tasks[:i], tasks[i+1:]...)
			return c.NoContent(http.StatusNoContent)
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{"message": "タスクが見つかりません"})
}

func main() {
	// Echoインスタンスの作成
	e := echo.New()

	// ルーティング定義
	e.GET("/tasks", getTasks)
	e.POST("/tasks", createTask)
	e.GET("/tasks/:id", getTask)
	e.PUT("/tasks/:id", updateTask)
	e.DELETE("/tasks/:id", deleteTask)

	// サーバーをポート8080で起動
	e.Logger.Fatal(e.Start(":8080"))
}
