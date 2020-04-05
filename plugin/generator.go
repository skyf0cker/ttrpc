/*
  Copyright The containerd Authors.

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

package plugin

import (
	"strings"

	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

type ttrpcGenerator struct {
	*generator.Generator
	generator.PluginImports

	typeurlPkg generator.Single
	ttrpcPkg   generator.Single
	contextPkg generator.Single
	protoPkg   generator.Single
	netPkg     generator.Single
}

func init() {
	generator.RegisterPlugin(new(ttrpcGenerator))
}

func (p *ttrpcGenerator) Name() string {
	return "ttrpc"
}

func (p *ttrpcGenerator) Init(g *generator.Generator) {
	p.Generator = g
}

func (p *ttrpcGenerator) Generate(file *generator.FileDescriptor) {
	p.PluginImports = generator.NewPluginImports(p.Generator)
	p.contextPkg = p.NewImport("context")
	p.typeurlPkg = p.NewImport("github.com/containerd/typeurl")
	p.ttrpcPkg = p.NewImport("github.com/containerd/ttrpc")
	p.protoPkg = p.NewImport("github.com/gogo/protobuf/proto")
	p.netPkg = p.NewImport("net")

	for _, service := range file.GetService() {
		serviceName := service.GetName()
		if pkg := file.GetPackage(); pkg != "" {
			serviceName = pkg + "." + serviceName
		}

		p.genService(serviceName, service)
	}
}

func (p *ttrpcGenerator) genService(fullName string, service *descriptor.ServiceDescriptorProto) {
	serviceName := service.GetName() + "Service"
	p.P()
	p.P("type ", serviceName, " interface{")
	p.In()
	for _, method := range service.Method {
		if method.GetClientStreaming() && method.GetServerStreaming() {
			p.P(method.GetName(), "(req ", service.GetName(), "_", method.GetName(), "Server) error")
			continue
		}
		p.P(method.GetName(),
			"(ctx ", p.contextPkg.Use(), ".Context, ",
			"req *", p.typeName(method.GetInputType()), ") ",
			"(*", p.typeName(method.GetOutputType()), ", error)")

	}
	p.Out()
	p.P("}")

	// client service interface
	p.P()
	p.P("type ", service.GetName(), "Client interface {")
	p.In()
	for _, method := range service.Method {
		if method.GetClientStreaming() && method.GetServerStreaming() {
			p.P(method.GetName(), "(ctx ", p.contextPkg.Use(), ".Context) ", service.GetName(), "_", method.GetName(), "Client")
			continue
		}
		p.P(method.GetName(),
			"(ctx ", p.contextPkg.Use(), ".Context, ",
			"req *", p.typeName(method.GetInputType()), ") ",
			"(*", p.typeName(method.GetOutputType()), ", error)")

	}
	p.Out()
	p.P("}")

	p.P()
	// registration method
	p.P("func Register", serviceName, "(srv *", p.ttrpcPkg.Use(), ".Server, svc ", serviceName, ") {")
	p.In()
	p.P(`srv.Register("`, fullName, `", `, p.ttrpcPkg.Use(), ".MethodSet{")
	p.In()
	p.P("UnaryMethod: map[string]", p.ttrpcPkg.Use(), ".Method{")
	p.In()
	for _, method := range service.Method {
		if method.GetServerStreaming() && method.GetClientStreaming() {
			continue
		}
		p.P(`"`, method.GetName(), `": `, `func(ctx context.Context, unmarshal func(interface{}) error) (interface{}, error) {`)
		p.In()
		p.P("var req ", p.typeName(method.GetInputType()))
		p.P(`if err := unmarshal(&req); err != nil {`)
		p.In()
		p.P(`return nil, err`)
		p.Out()
		p.P(`}`)
		p.P("return svc.", method.GetName(), "(ctx, &req)")
		p.Out()
		p.P("},")
	}
	p.Out()
	p.P("},")
	p.P("StreamMethod: map[string]", p.ttrpcPkg.Use(), ".StreamMethod{")
	p.In()
	for _, method := range service.Method {
		if method.GetServerStreaming() && method.GetClientStreaming() {
			p.P(`"`, method.GetName(), `": `, `func(ctx context.Context, conn `, p.netPkg.Use(), `.Conn, srvStream *`, p.ttrpcPkg.Use(), `.ServerStream) error {`)
			p.In()
			p.P("stream := &", service.GetName(), method.GetName(), "Server{srvStream}")
			p.P("return svc.", method.GetName(), "(stream)")
			p.Out()
			p.P("},")
		}
	}
	p.Out()
	p.P("},")
	p.Out()
	p.P("})")
	p.Out()
	p.P("}")

	clientType := service.GetName() + "Client"
	clientStructType := strings.ToLower(clientType[:1]) + clientType[1:]
	p.P()
	p.P("type ", clientStructType, " struct{")
	p.In()
	p.P("client *", p.ttrpcPkg.Use(), ".Client")
	p.Out()
	p.P("}")
	p.P()
	p.P("func New", clientType, "(client *", p.ttrpcPkg.Use(), ".Client) ", service.GetName(), "Client {")
	p.In()
	p.P("return &", clientStructType, "{")
	p.In()
	p.P("client: client,")
	p.Out()
	p.P("}")
	p.Out()
	p.P("}")
	p.P()
	for _, method := range service.Method {
		if method.GetServerStreaming() && method.GetClientStreaming() {
			p.P("func (c *", clientStructType, ") ", method.GetName(), "(ctx ", p.contextPkg.Use(), ".Context )", service.GetName(), "_", method.GetName(), "Client {")
			p.In()
			p.P("stream := ", p.ttrpcPkg.Use(), `.NewClientStream(c.client.GetConn(), c.client.GetCalls(), "`, fullName, `", "`, method.GetName(), `")`)
			p.P("clientStream := &", service.GetName(), method.GetName(), "Client{stream}")
			p.P("return clientStream")
			p.Out()
			p.P("}")
			continue
		}

		p.P()
		p.P("func (c *", clientStructType, ") ", method.GetName(),
			"(ctx ", p.contextPkg.Use(), ".Context, ",
			"req *", p.typeName(method.GetInputType()), ") ",
			"(*", p.typeName(method.GetOutputType()), ", error) {")
		p.In()
		p.P("var resp ", p.typeName(method.GetOutputType()))
		p.P("if err := c.client.Call(ctx, ", `"`+fullName+`", `, `"`+method.GetName()+`", `, p.ttrpcPkg.Use(), ".MessageTypeRequest, req, &resp); err != nil {")
		p.In()
		p.P("return nil, err")
		p.Out()
		p.P("}")
		p.P("return &resp, nil")
		p.Out()
		p.P("}")
	}

	for _, method := range service.Method {
		p.P()

		if method.GetClientStreaming() && method.GetServerStreaming() {
			// server interfacev implement
			p.P("type ", service.GetName(), "_", method.GetName(), "Server interface {")
			p.In()
			p.P("Send(*", p.typeName(method.GetOutputType()), ") error")
			p.P("Recv() (*", p.typeName(method.GetInputType()), ", error)")
			//p.P(p.ttrpcPkg.Use(), ".StreamInf")
			p.Out()
			p.P("}")

			p.P("type ", service.GetName(), method.GetName(), "Server struct {")
			p.In()
			p.P("Stream ", p.ttrpcPkg.Use(), ".StreamInf")
			p.Out()
			p.P("}")

			p.P()
			p.P("func (x *", service.GetName(), method.GetName(), "Server) Send(m *", p.typeName(method.GetOutputType()), ") error {")
			p.In()
			p.P("payload, err := ", p.protoPkg.Use(), ".Marshal(m)")
			p.P("if err != nil {")
			p.In()
			p.P("return err")
			p.Out()
			p.P("}")
			p.P("x.Stream.SendMsg(payload)")
			p.P("return err")
			p.Out()
			p.P("}")

			p.P("func (x *", service.GetName(), method.GetName(), "Server) Recv() (*", p.typeName(method.GetInputType()), ", error) {")
			p.In()
			p.P("payload, err := x.Stream.RecvMsg()")
			p.P("if err != nil {")
			p.In()
			p.P("return nil, err")
			p.Out()
			p.P("}")
			p.P()
			p.P("msg := &", p.typeName(method.GetInputType()), "{}")
			p.P("marshalErr := ", p.protoPkg.Use(), ".Unmarshal(payload, msg)")
			p.P("if marshalErr != nil {")
			p.In()
			p.P("return nil, marshalErr")
			p.Out()
			p.P("}")
			p.P("return msg, nil")
			p.Out()
			p.P("}")

			// client interface
			p.P("type ", service.GetName(), "_", method.GetName(), "Client interface {")
			p.In()
			p.P("Send(*", p.typeName(method.GetInputType()), ") error")
			p.P("Recv() (*", p.typeName(method.GetOutputType()), ", error)")
			//p.P(p.ttrpcPkg.Use(), ".StreamInf")
			p.Out()
			p.P("}")

			p.P("type ", service.GetName(), method.GetName(), "Client struct {")
			p.In()
			p.P("Stream ", p.ttrpcPkg.Use(), ".StreamInf")
			p.Out()
			p.P("}")

			p.P("func (x *", service.GetName(), method.GetName(), "Client) Send(m *", p.typeName(method.GetInputType()), ") error {")
			p.In()
			p.P("payload, err := ", p.protoPkg.Use(), ".Marshal(m)")
			p.P("if err != nil {")
			p.In()
			p.P("return err")
			p.Out()
			p.P("}")
			p.P("x.Stream.SendMsg(payload)")
			p.P("return err")
			p.Out()
			p.P("}")

			p.P("func (x *", service.GetName(), method.GetName(), "Client) Recv() (*", p.typeName(method.GetOutputType()), ", error) {")
			p.In()
			p.P("payload, err := x.Stream.RecvMsg()")
			p.P("if err != nil {")
			p.In()
			p.P("return nil, err")
			p.Out()
			p.P("}")
			p.P()
			p.P("msg := &", p.typeName(method.GetOutputType()), "{}")
			p.P("marshalErr := ", p.protoPkg.Use(), ".Unmarshal(payload, msg)")
			p.P("if marshalErr != nil {")
			p.In()
			p.P("return nil, marshalErr")
			p.Out()
			p.P("}")
			p.P("return msg, nil")
			p.Out()
			p.P("}")
		}
	}
}

func (p *ttrpcGenerator) objectNamed(name string) generator.Object {
	p.Generator.RecordTypeUse(name)
	return p.Generator.ObjectNamed(name)
}

func (p *ttrpcGenerator) typeName(str string) string {
	return p.Generator.TypeName(p.objectNamed(str))
}
