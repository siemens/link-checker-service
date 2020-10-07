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

var awsLambdaCmd = &cobra.Command{
	Use:   "awslambda",
	Short: "start the service as an AWS Lambda",
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
}
