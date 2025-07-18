module example

go 1.23.1

require (
	github.com/JohnPlummer/reddit-client v0.0.0
	github.com/joho/godotenv v1.5.1
)

require golang.org/x/time v0.5.0 // indirect

replace github.com/JohnPlummer/reddit-client => ../../
