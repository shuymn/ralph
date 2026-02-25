package main

import "io"

func RunReview(root string, stdout, stderr io.Writer) int {
	return runCommand(root, stdout, stderr, commandReview, false)
}

func RunReviewDry(root string, stdout, stderr io.Writer) int {
	return runCommand(root, stdout, stderr, commandReview, true)
}
