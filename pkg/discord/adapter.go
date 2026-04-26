// Package discord provides a Discord channel adapter for Bonsai Brain.
// It turns Discord messages into agent interactions and logs everything
// to designated channels.
package discord

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/middleware"
	"github.com/donn/bonsai-brain/pkg/swarm"
)
