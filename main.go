package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mg MongoInstance

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    int     `json:"age"`
}

var mongoURL string

func Connect() error {
	err := godotenv.Load()
	if err != nil {
		return err
	}
	USER := os.Getenv("USER_ID")
	userPassword := os.Getenv("USER_PASSWORD")

	const dbName = "fiber-hrms"

	mongoURL = "mongodb+srv://" + USER + ":" + userPassword + "@cluster0.sqm6wpi.mongodb.net/" + dbName + "?retryWrites=true&w=majority&appName=Cluster0"

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		return err
	}

	// MongoDB에 연결
	db := client.Database(dbName)

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}
	return nil
}
func getHandler(c *fiber.Ctx) error {
	collection := mg.Db.Collection("employees")
	cursor, err := collection.Find(c.Context(), bson.D{})
	if err != nil {
		return err
	}
	defer cursor.Close(c.Context())

	var employees []Employee
	if err := cursor.All(c.Context(), &employees); err != nil {
		return err
	}

	return c.JSON(employees)
}

func PostEmployee(c *fiber.Ctx) error {
	collection := mg.Db.Collection("employees")

	employee := new(Employee)

	if err := c.BodyParser(employee); err != nil {
		return err
	}

	employee.ID = ""

	insertionResult, err := collection.InsertOne(c.Context(), employee)
	if err != nil {
		return err
	}

	filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
	createdRecord := collection.FindOne(c.Context(), filter)

	createdEmployee := &Employee{}
	if err := createdRecord.Decode(createdEmployee); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(createdEmployee)
}
func main() {
	if err := Connect(); err != nil {
		fmt.Println(err)
		return
	}

	app := fiber.New()

	app.Get("/employee", getHandler)
	app.Post("/employee", PostEmployee)

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")

		employeeId, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.SendStatus(400)
		}

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeId}}

		update := bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.SendStatus(500)
		}
		employee.ID = idParam
		return c.Status(200).JSON(employee)
	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		employeeId, err := primitive.ObjectIDFromHex(c.Params("id"))

		if err != nil {
			return c.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: employeeId}}
		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)

		if err != nil {
			return c.SendStatus(500)
		}
		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}
		return c.Status(200).JSON("record deleted")

	})

	log.Fatal(app.Listen(":3000"))

}
