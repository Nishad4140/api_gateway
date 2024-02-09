package graph

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/Nishad4140/api_gateway/middleware"
	"github.com/Nishad4140/proto_files/pb"
	"github.com/graphql-go/graphql"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	Secret       []byte
	ProductsConn pb.ProductServiceClient
	UsersConn    pb.UserServiceClient
)

func RetrieveSecret(secretString string) {
	Secret = []byte(secretString)
}

func Initialize(prodConn pb.ProductServiceClient, userConn pb.UserServiceClient) {
	ProductsConn = prodConn
	UsersConn = userConn
}

var ProductType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "product",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"price": &graphql.Field{
				Type: graphql.Int,
			},
			"quantity": &graphql.Field{
				Type: graphql.Int,
			},
		},
	},
)

var UserType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "user",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"email": &graphql.Field{
				Type: graphql.String,
			},
			"password": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

var RootQuery = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			"product": &graphql.Field{
				Type: graphql.NewList(ProductType),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return ProductsConn.GetProduct(context.Background(), &pb.GetProductByID{
						Id: uint32(p.Args["id"].(int)),
					})
				},
			},
			"products": &graphql.Field{
				Type: graphql.NewList(ProductType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					var res []*pb.AddProductResponse

					products, err := ProductsConn.GetAllProducts(context.Background(), &emptypb.Empty{})
					if err != nil {
						fmt.Println(err.Error())
					}

					for {
						prod, err := products.Recv()
						if err == io.EOF {
							break
						}
						if err != nil {
							fmt.Println(err)
						}
						res = append(res, prod)
					}
					return res, err
				},
			},
		},
	},
)

var Mutation = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"UserSignUp": &graphql.Field{
				Type: UserType,
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"password": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					userData, err := UsersConn.UserSignUp(context.Background(), &pb.UserSignUpRequest{
						Name:     p.Args["name"].(string),
						Email:    p.Args["email"].(string),
						Password: p.Args["password"].(string),
					})
					if err != nil {
						fmt.Println(err.Error())
						return nil, err
					}
					return userData, nil
				},
			},
			"AddProduct": &graphql.Field{
				Type: ProductType,
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"price": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
					"quantity": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: middleware.AdminMiddleware(func(p graphql.ResolveParams) (interface{}, error) {

					fmt.Println("here reached...")
					products, err := ProductsConn.AddProduct(context.Background(), &pb.AddProductRequest{
						Name:     p.Args["name"].(string),
						Price:    int32(p.Args["price"].(int)),
						Quantity: int32(p.Args["quantity"].(int)),
					})
					if err != nil {
						fmt.Println(err.Error())
						return nil, err
					}
					return products, nil
				}),
			},
			"UpdateStock": &graphql.Field{
				Type: ProductType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"stock": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
					"increase": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Boolean),
					},
				},
				Resolve: middleware.AdminMiddleware(func(p graphql.ResolveParams) (interface{}, error) {
					id, _ := strconv.Atoi(p.Args["id"].(string))
					return ProductsConn.UpdateStock(context.Background(), &pb.UpdateStockRequest{
						Id:       uint32(id),
						Quantity: int32(p.Args["stock"].(int)),
						Increase: p.Args["increase"].(bool),
					})
				}),
			},
		},
	},
)

var Schema, _ = graphql.NewSchema(graphql.SchemaConfig{
	Query:    RootQuery,
	Mutation: Mutation,
})
