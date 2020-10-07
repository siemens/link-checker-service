// Copyright 2020 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	s "github.com/siemens/link-checker-service/server"
	"github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/spf13/cobra"
)

var ginLambda *ginadapter.GinLambda

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return ginLambda.ProxyWithContext(ctx, req)
}

// awsLambdaCmd represents the awsLambda command
var awsLambdaCmd = &cobra.Command{
	Use:   "awslambda",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("awsLambda called")
		fetchConfig()
		echoConfig()
		server := s.NewServerWithOptions(&s.Options{
			CORSOrigins:           corsOrigins,
			IPRateLimit:           IPRateLimit,
			MaxURLsInRequest:      maxURLsInRequest,
			DisableRequestLogging: true,
			DomainBlacklistGlobs:  domainBlacklistGlobs,
		})
		ginLambda = ginadapter.New(server.Detail())
		lambda.Start(Handler)
	},
}

func init() {
	serveCmd.AddCommand(awsLambdaCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// awsLambdaCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// awsLambdaCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
