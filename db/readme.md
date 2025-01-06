# Как пользоваться базой

### Подключаемся

```go
conn, err := db.NewDb(db.Config{Dsn: "DSN_STRING"})
if err != nil {...}
```

### Создание объекта

```go
config := &models.ProblemConfig{ProblemType: models.ProblemType_ICPC}
result := conn.Create(&config)
config.ID  // номер нового объекта
result.Error  // если была ошибка
result.RowsAffected  // сколько строк вставилосб
```

https://gorm.io/docs/create.html

### Получение объекта (обычно одного)

```go
var problem Problem
var problems []Problem
err := conn.First(&problem)
// теперь либо err.Error != nil, либо в problem лежит запись с минимальным id
conn.Take(&problem)  // рандомная запись
conn.Last(&problem)  // последняя запись
errors.Is(result.Error, gorm.ErrRecordNotFound) // пример ошибки

result := map[string]interface{}{}
conn.Model(&models.Problem{}).First(&result)  // после этого можно обращаться как к мапе

conn.Find(&problem, 1)  // выбрать с id == 1
conn.Find(&problem, "id = ?", 1)
conn.Find(&problems, []int{1, 2, 3})  // выбрать с id \in {1, 2, 3}
conn.Find(&problems)  // все объекты
```

https://gorm.io/docs/query.html

### Условия

```go
// все IOI задачи
conn.Where(models.Problem{ProblemType: models.ProblemType_IOI}).Find(&problems)
conn.Find(&problems, "problem_type = ?", models.ProblemType_IOI)
conn.Find(&problems, map[string]any{"problem_type": models.ProblemType_IOI})
conn.Not("problem_type = ?", models.ProblemType_ICPC).Find(&problems)
// все IOI задачи или с id == 1
conn. 
    Where("problem_type = ?", models.ProblemType_IOI).
    Or(1).
    Find(&problems)
```
