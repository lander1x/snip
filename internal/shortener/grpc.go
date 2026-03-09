package shortener

import (
	"context"
	"errors"

	pb "github.com/landerix/snip/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	pb.UnimplementedShortenerServer
	svc *Service
}

func NewGRPCServer(svc *Service) *GRPCServer {
	return &GRPCServer{svc: svc}
}

func (s *GRPCServer) CreateLink(ctx context.Context, req *pb.CreateLinkRequest) (*pb.Link, error) {
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	link, err := s.svc.CreateLink(ctx, req.Url, req.CustomAlias)
	if err != nil {
		if errors.Is(err, ErrInvalidURL) || errors.Is(err, ErrInvalidAlias) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoLink(link, s.svc.ShortURL(link.Code)), nil
}

func (s *GRPCServer) ResolveLink(ctx context.Context, req *pb.ResolveLinkRequest) (*pb.Link, error) {
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	link, err := s.svc.ResolveLink(ctx, req.Code)
	if err != nil {
		if errors.Is(err, ErrLinkNotFound) {
			return nil, status.Error(codes.NotFound, "link not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toProtoLink(link, s.svc.ShortURL(link.Code)), nil
}

func (s *GRPCServer) DeleteLink(ctx context.Context, req *pb.DeleteLinkRequest) (*emptypb.Empty, error) {
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	err := s.svc.DeleteLink(ctx, req.Code)
	if err != nil {
		if errors.Is(err, ErrLinkNotFound) {
			return nil, status.Error(codes.NotFound, "link not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func toProtoLink(link *Link, shortURL string) *pb.Link {
	return &pb.Link{
		Id:        link.ID.String(),
		Code:      link.Code,
		Url:       link.URL,
		ShortUrl:  shortURL,
		Clicks:    link.Clicks,
		CreatedAt: timestamppb.New(link.CreatedAt),
	}
}
