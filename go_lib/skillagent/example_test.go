// Package skillagent example shows how to use the skillagent SDK.
//
// This file demonstrates the basic usage patterns for the SkillAgent SDK,
// including skill discovery, execution, and streaming.
package skillagent_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"go_lib/skillagent"

	"github.com/cloudwego/eino/components/model"
)

// ExampleNewSkillAgent demonstrates how to create a SkillAgent.
func ExampleNewSkillAgent() {
	ctx := context.Background()

	// You need to create a ChatModel that supports tool calling.
	// This example shows the pattern, but you need to provide your own model.
	//
	// Example with OpenAI:
	//   chatModel, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
	//       APIKey: "your-api-key",
	//       Model:  "gpt-4",
	//   })

	var chatModel model.ChatModel // Replace with actual model

	// Create the SkillAgent with ADK middleware architecture
	agent, err := skillagent.NewSkillAgent(ctx, &skillagent.SkillAgentConfig{
		ChatModel: chatModel,
		SkillsDir: "/path/to/skills",
		MaxStep:   50,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer agent.Close()

	// List discovered skills (ADK skill middleware handles discovery automatically)
	skills, err := agent.ListSkills(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, skill := range skills {
		fmt.Printf("Skill: %s - %s\n", skill.Name, skill.Description)
	}
}

// ExampleSkillAgent_Execute demonstrates how to execute a task.
func ExampleSkillAgent_Execute() {
	ctx := context.Background()

	// Assume agent is already created
	var agent *skillagent.SkillAgent

	// Execute a task - the agent will automatically select the best skill
	result, err := agent.Execute(ctx, "Generate a weekly report for the sales team")
	if err != nil {
		log.Fatal(err)
	}

	if result.Success {
		fmt.Println("Output:", result.Output)
		fmt.Println("Used skill:", result.ActivatedSkill)
		fmt.Println("Duration:", result.Duration)
	} else {
		fmt.Println("Execution failed:", result.Error)
	}
}

// ExampleSkillAgent_ExecuteWithSkill demonstrates executing with a specific skill.
func ExampleSkillAgent_ExecuteWithSkill() {
	ctx := context.Background()
	var agent *skillagent.SkillAgent

	// Force execution with a specific skill
	result, err := agent.Execute(ctx, "Generate report",
		skillagent.WithForceSkill("report-generator"),
		skillagent.WithTimeout(5*time.Minute),
		skillagent.WithVariables(map[string]interface{}{
			"department": "sales",
			"period":     "weekly",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Output)
}

// ExampleSkillAgent_ExecuteStream demonstrates streaming execution.
func ExampleSkillAgent_ExecuteStream() {
	ctx := context.Background()
	var agent *skillagent.SkillAgent

	// Execute with streaming
	eventCh, err := agent.ExecuteStream(ctx, "Analyze the code for security issues")
	if err != nil {
		log.Fatal(err)
	}

	// Process events as they arrive
	for event := range eventCh {
		switch event.Type {
		case skillagent.StreamEventSkillSelected:
			data := event.Data.(skillagent.SkillSelectedData)
			fmt.Printf("Selected skill: %s\n", data.SkillName)

		case skillagent.StreamEventToolCalling:
			data := event.Data.(skillagent.ToolCallingData)
			fmt.Printf("Calling tool: %s\n", data.ToolName)

		case skillagent.StreamEventPartialOutput:
			data := event.Data.(skillagent.PartialOutputData)
			fmt.Print(data.Content)

		case skillagent.StreamEventFinalOutput:
			data := event.Data.(skillagent.FinalOutputData)
			fmt.Println("\n--- Final Output ---")
			fmt.Println(data.Content)

		case skillagent.StreamEventError:
			fmt.Printf("Error: %v\n", event.Error)

		case skillagent.StreamEventComplete:
			data := event.Data.(skillagent.CompleteData)
			fmt.Printf("\nCompleted in %v with %d tool calls\n",
				data.Duration, data.ToolCallCount)
		}
	}
}

// ExampleSkillHooks demonstrates using lifecycle hooks.
func ExampleSkillHooks() {
	ctx := context.Background()
	var chatModel model.ChatModel

	_, _ = skillagent.NewSkillAgent(ctx, &skillagent.SkillAgentConfig{
		ChatModel: chatModel,
		SkillsDir: "/path/to/skills",
		Hooks: &skillagent.SkillHooks{
			OnSkillDiscovered: func(metadata *skillagent.SkillMetadata) {
				log.Printf("Discovered skill: %s", metadata.Name)
			},
			OnSkillActivated: func(manifest *skillagent.SkillManifest) {
				log.Printf("Activated skill: %s (v%s)", manifest.Name, manifest.Version)
			},
			OnToolCall: func(toolName string, arguments string) {
				log.Printf("Tool call: %s(%s)", toolName, arguments)
			},
			OnComplete: func(result *skillagent.ExecutionResult) {
				log.Printf("Execution completed: success=%v, skill=%s",
					result.Success, result.ActivatedSkill)
			},
		},
	})
}

// Example_skillDirectory shows the expected skill directory structure.
func Example_skillDirectory() {
	// A skill is a directory containing a SKILL.md file:
	//
	// skills/
	// └── report-generator/
	//     ├── SKILL.md              # Required: metadata + instructions
	//     ├── scripts/              # Optional: executable scripts
	//     │   └── generate.py
	//     ├── templates/            # Optional: template files
	//     │   └── report.md
	//     ├── references/           # Optional: reference documents
	//     │   └── style-guide.md
	//     └── assets/               # Optional: images, data files
	//         └── logo.png
	//
	// The SKILL.md file must have this format:
	//
	// ---
	// name: report-generator
	// description: Generate formatted reports from data
	// version: 1.0.0
	// allowed-tools:              # Optional: restrict available tools
	//   - read_skill_file
	//   - execute_script
	// ---
	//
	// # Report Generator
	//
	// ## When to use
	// Use this skill when the user wants to generate reports...
	//
	// ## Instructions
	// 1. First, read the template from templates/report.md
	// 2. Then, execute scripts/generate.py with the data
	// 3. Format the output according to the template
}
