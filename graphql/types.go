package graph

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Nishad4140/api_gateway/authorize"
	"github.com/Nishad4140/api_gateway/middleware"
	"github.com/Nishad4140/proto_files/pb"
	"github.com/graphql-go/graphql"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	Secret       []byte
	ProductsConn pb.ProductServiceClient
	UsersConn    pb.UserServiceClient
	CartConn     pb.CartServiceClient
	OrderConn    pb.OrderServiceClient
)

func RetrieveSecret(secretString string) {
	Secret = []byte(secretString)
}

func Initialize(prodConn pb.ProductServiceClient, userConn pb.UserServiceClient, cartConn pb.CartServiceClient, orderConn pb.OrderServiceClient) {
	ProductsConn = prodConn
	UsersConn = userConn
	CartConn = cartConn
	OrderConn = orderConn
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

var CartType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "cart",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"userId": &graphql.Field{
				Type: graphql.Int,
			},
			"productId": &graphql.Field{
				Type: graphql.Int,
			},
			"quantity": &graphql.Field{
				Type: graphql.Int,
			},
			"total": &graphql.Field{
				Type: graphql.Float,
			},
		},
	},
)

var OrderType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "order",
		Fields: graphql.Fields{
			"orderId": &graphql.Field{
				Type: graphql.Int,
			},
			"orderItems": &graphql.Field{
				Type: graphql.NewList(ProductType),
			},
			"addressId": &graphql.Field{
				Type: graphql.Int,
			},
			"orderStatusId": &graphql.Field{
				Type: graphql.Int,
			},
			"paymentTypeId": &graphql.Field{
				Type: graphql.Int,
			},
			"total": &graphql.Field{
				Type: graphql.Float,
			},
		},
	},
)

var RootQuery = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			"userlogin": &graphql.Field{
				Type: UserType,
				Args: graphql.FieldConfigArgument{
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"password": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					res, err := UsersConn.UserLogin(context.Background(), &pb.LoginRequest{
						Email:    p.Args["email"].(string),
						Password: p.Args["password"].(string),
					})
					if err != nil {
						return nil, err
					}
					fmt.Println("before token")
					token, err := authorize.GenerateJwt(uint(res.Id), false, false, Secret)
					if err != nil {
						fmt.Println("error here:", err)
						return nil, err
					}

					w := p.Context.Value("httpResponseWriter").(http.ResponseWriter)

					fmt.Println("after token")

					http.SetCookie(w, &http.Cookie{
						Name:  "jwtToken",
						Value: token,
						Path:  "/",
					})

					return res, nil
				},
			},
			"adminlogin": &graphql.Field{
				Type: UserType,
				Args: graphql.FieldConfigArgument{
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"password": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					res, err := UsersConn.AdminLogin(context.Background(), &pb.LoginRequest{
						Email:    p.Args["email"].(string),
						Password: p.Args["password"].(string),
					})
					if err != nil {
						return nil, err
					}
					token, err := authorize.GenerateJwt(uint(res.Id), true, false, Secret)
					if err != nil {
						return nil, err
					}

					w := p.Context.Value("httpResponseWriter").(http.ResponseWriter)

					http.SetCookie(w, &http.Cookie{
						Name:  "jwtToken",
						Value: token,
						Path:  "/",
					})
					return res, nil
				},
			},
			"supadminlogin": &graphql.Field{
				Type: UserType,
				Args: graphql.FieldConfigArgument{
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"password": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					res, err := UsersConn.SupAdminLogin(context.Background(), &pb.LoginRequest{
						Email:    p.Args["email"].(string),
						Password: p.Args["password"].(string),
					})
					if err != nil {
						return nil, err
					}
					token, err := authorize.GenerateJwt(uint(res.Id), true, true, Secret)
					if err != nil {
						return nil, err
					}

					w := p.Context.Value("httpResponseWriter").(http.ResponseWriter)

					http.SetCookie(w, &http.Cookie{
						Name:  "jwtToken",
						Value: token,
						Path:  "/",
					})
					return res, nil
				},
			},
			"GetAllAdmins": &graphql.Field{
				Type: graphql.NewList(UserType),
				Resolve: middleware.SupAdminMiddleware(func(p graphql.ResolveParams) (interface{}, error) {
					admins, err := UsersConn.GetAllAdmins(context.Background(), &emptypb.Empty{})
					if err != nil {
						return nil, err
					}
					var res []*pb.UserResponse
					for {
						admin, err := admins.Recv()
						if err == io.EOF {
							break
						}
						fmt.Println(admin.Name)
						if err != nil {
							fmt.Println(err.Error())
						}
						res = append(res, admin)
					}
					fmt.Println(res)
					return res, nil
				}),
			},
			"GetAllUsers": &graphql.Field{
				Type: graphql.NewList(UserType),
				Resolve: middleware.AdminMiddleware(func(p graphql.ResolveParams) (interface{}, error) {
					users, err := UsersConn.GetAllUsers(context.Background(), &emptypb.Empty{})
					if err != nil {
						return nil, err
					}
					var res []*pb.UserResponse
					for {
						user, err := users.Recv()
						if err == io.EOF {
							break
						}
						if err != nil {
							fmt.Println(err.Error())
						}
						res = append(res, user)

					}
					return res, nil
				}),
			},
			"product": &graphql.Field{
				Type: ProductType,
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
			"GetAllCartItems": &graphql.Field{
				Type: graphql.NewList(CartType),
				Resolve: middleware.UserMiddleware(func(p graphql.ResolveParams) (interface{}, error) {
					userId := p.Context.Value("userId").(uint)
					cartItems, err := CartConn.GetAllCart(context.Background(), &pb.CartCreate{
						UserId: uint32(userId),
					})
					if err != nil {
						return nil, err
					}
					var res []*pb.GetAllCartResponse
					for {
						item, err := cartItems.Recv()
						if err == io.EOF {
							break
						}
						if err != nil {
							fmt.Println(err.Error())
						}
						res = append(res, item)
					}
					return res, nil
				}),
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
					// userData, err := UsersConn.UserSignUp(context.Background(), &pb.UserSignUpRequest{
					// 	Name:     p.Args["name"].(string),
					// 	Email:    p.Args["email"].(string),
					// 	Password: p.Args["password"].(string),
					// })
					// if err != nil {
					// 	fmt.Println(err.Error())
					// 	return nil, err
					// }
					// return userData, nil
					name, _ := p.Args["name"].(string)
					email, _ := p.Args["email"].(string)
					password, _ := p.Args["password"].(string)

					if name == "" || email == "" || password == "" {
						return nil, fmt.Errorf("name, email, and password are required")
					}
					res, err := UsersConn.UserSignUp(context.Background(), &pb.UserSignUpRequest{
						Name:     p.Args["name"].(string),
						Email:    p.Args["email"].(string),
						Password: p.Args["password"].(string),
					})
					if err != nil {
						return nil, err
					}
					fmt.Println("befor cart")
					cart, err := CartConn.CreateCart(context.Background(), &pb.CartCreate{
						UserId: res.Id,
					})
					if err != nil {
						return nil, err
					}
					if cart.UserId == 0 {
						return nil, fmt.Errorf("error while creating cart")
					}
					fmt.Println("cart user id", cart.UserId, cart.CartId)
					response := &pb.UserResponse{
						Id:    res.Id,
						Name:  res.Name,
						Email: res.Email,
					}
					return response, nil
				},
			},
			"addAdmin": &graphql.Field{
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
				Resolve: middleware.SupAdminMiddleware(func(p graphql.ResolveParams) (interface{}, error) {

					admin, err := UsersConn.AddAdmin(context.Background(), &pb.UserSignUpRequest{
						Name:     p.Args["name"].(string),
						Email:    p.Args["email"].(string),
						Password: p.Args["password"].(string),
					})
					fmt.Println(admin)
					if err != nil {
						fmt.Println(err.Error())
						return nil, err
					}

					return admin, nil
				}),
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
			"AddToCart": &graphql.Field{
				Type: CartType,
				Args: graphql.FieldConfigArgument{
					"productId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
					"quantity": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: middleware.UserMiddleware(func(p graphql.ResolveParams) (interface{}, error) {
					userIDval := p.Context.Value("userId").(uint)
					res, err := CartConn.AddToCart(context.Background(), &pb.AddToCartRequest{
						UserId:   uint32(userIDval),
						ProdId:   uint32(p.Args["productId"].(int)),
						Quantity: int32(p.Args["quantity"].(int)),
					})
					if err != nil {
						return nil, err
					}
					fmt.Println(res)
					return res, nil
				}),
			},
			"RemoveFromCart": &graphql.Field{
				Type: CartType,
				Args: graphql.FieldConfigArgument{
					"productId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: middleware.UserMiddleware(func(p graphql.ResolveParams) (interface{}, error) {
					userId := p.Context.Value("userId").(uint)
					return CartConn.RemoveCart(context.Background(), &pb.RemoveCartRequest{
						UserId: uint32(userId),
						ProdId: uint32(p.Args["productId"].(int)),
					})
				}),
			},
			"OrderAll": &graphql.Field{
				Type: OrderType,
				Resolve: middleware.UserMiddleware(func(p graphql.ResolveParams) (interface{}, error) {
					userId := p.Context.Value("userId").(uint)
					order, err := OrderConn.OrderAll(context.Background(), &pb.UserId{
						UserId: uint32(userId),
					})
					if err != nil {
						return nil, err
					}
					orderMap := map[string]interface{}{
						"id": order.OrderId,
					}
					return orderMap, nil
				}),
			},
			"CancelOrder": &graphql.Field{
				Type: OrderType,
				Args: graphql.FieldConfigArgument{
					"orderId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: middleware.UserMiddleware(func(p graphql.ResolveParams) (interface{}, error) {
					return OrderConn.CancelOrder(context.Background(), &pb.OrderId{
						OrderId: uint32(p.Args["orderId"].(int)),
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
