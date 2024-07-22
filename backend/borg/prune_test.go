package borg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*

TEST CASES - prune.go

* parsePruneReason
* parsePruneName
* parsePruneOutput
// TODO: decodePruneOutput

*/

var _ = Describe("prune.go", Ordered, func() {
	type inputOutput struct {
		input  string
		output string
	}

	It("parsePruneReason", func() {
		// ARRANGE
		inputOutput := []inputOutput{
			{
				input:  "Keeping archive (rule: daily #1):            down-2024-07-22-21-19-22             Mon, 2024-07-22 21:19:23 [c8396fd6b334e09fae91d213039f91533b047bb22103a30613443ed7cdfc4056]",
				output: "daily #1",
			},
			{
				input:  "Would prune:                                 down-2024-07-22-21-19-20             Mon, 2024-07-22 21:19:21 [36535bbf6b2e563e805c73be8827d4c648cc39e2ad9eb82fa8b097ff52899019]",
				output: "",
			},
			{
				input:  "Keeping archive (rule: daily #2):            down-2024-07-21-18-16-03             Sun, 2024-07-21 18:16:04 [d22f69e1c874bff4d26a1d14440d1b8c0e12d968745145bb7f10e796a1912ff1]",
				output: "daily #2",
			},
			{
				input:  "Keeping archive (rule: daily[oldest] #3):    down-2024-07-21-16-21-11             Sun, 2024-07-21 16:21:12 [3dbb5b8b7eff848fb2fa3b4455ddc0af81fed0f125bd18e9799a065b014ab166]",
				output: "daily[oldest] #3",
			},
			{
				input:  "terminating with success status, rc 0",
				output: "",
			},
		}

		// ACT
		for _, test := range inputOutput {
			output := parsePruneReason(test.input)

			// ASSERT
			Expect(output).To(Equal(test.output))
		}
	})

	type logInputOutput struct {
		input   LogMessage
		output1 *PruneArchive
		output2 *KeepArchive
	}

	It("parsePruneName", func() {
		// ARRANGE
		inputOutput := []inputOutput{
			{
				input:  "Keeping archive (rule: daily #1):            down-2024-07-22-21-19-22             Mon, 2024-07-22 21:19:23 [c8396fd6b334e09fae91d213039f91533b047bb22103a30613443ed7cdfc4056]",
				output: "down-2024-07-22-21-19-22",
			},
			{
				input:  "Would prune:                                 down-2024-07-22-21-19-20             Mon, 2024-07-22 21:19:21 [36535bbf6b2e563e805c73be8827d4c648cc39e2ad9eb82fa8b097ff52899019]",
				output: "down-2024-07-22-21-19-20",
			},
			{
				input:  "Keeping archive (rule: daily #2):            down-2024-07-21-18-16-03             Sun, 2024-07-21 18:16:04 [d22f69e1c874bff4d26a1d14440d1b8c0e12d968745145bb7f10e796a1912ff1]",
				output: "down-2024-07-21-18-16-03",
			},
			{
				input:  "Keeping archive (rule: daily[oldest] #3):    down-2024-07-21-16-21-11             Sun, 2024-07-21 16:21:12 [3dbb5b8b7eff848fb2fa3b4455ddc0af81fed0f125bd18e9799a065b014ab166]",
				output: "down-2024-07-21-16-21-11",
			},
			{
				input:  "terminating with success status, rc 0",
				output: "",
			},
		}

		// ACT
		for _, test := range inputOutput {
			output := parsePruneName(test.input)

			// ASSERT
			Expect(output).To(Equal(test.output))
		}
	})

	It("parsePruneOutput", func() {
		// ARRANGE
		inputOutput := []logInputOutput{
			{
				input: LogMessage{
					Message: "Keeping archive (rule: daily #1):            down-2024-07-22-21-19-22             Mon, 2024-07-22 21:19:23 [c8396fd6b334e09fae91d213039f91533b047bb22103a30613443ed7cdfc4056]",
				},
				output1: nil,
				output2: &KeepArchive{
					Name:   "down-2024-07-22-21-19-22",
					Reason: "daily #1",
				},
			},
			{
				input: LogMessage{
					Message: "Would prune:                                 down-2024-07-22-21-19-20             Mon, 2024-07-22 21:19:21 [36535bbf6b2e563e805c73be8827d4c648cc39e2ad9eb82fa8b097ff52899019]",
				},
				output1: &PruneArchive{
					Name: "down-2024-07-22-21-19-20",
				},
				output2: nil,
			},
			{
				input: LogMessage{
					Message: "Keeping archive (rule: daily #2):            down-2024-07-21-18-16-03             Sun, 2024-07-21 18:16:04 [d22f69e1c874bff4d26a1d14440d1b8c0e12d968745145bb7f10e796a1912ff1]",
				},
				output1: nil,
				output2: &KeepArchive{
					Name:   "down-2024-07-21-18-16-03",
					Reason: "daily #2",
				},
			},
			{
				input: LogMessage{
					Message: "Keeping archive (rule: daily[oldest] #3):    down-2024-07-21-16-21-11             Sun, 2024-07-21 16:21:12 [3dbb5b8b7eff848fb2fa3b4455ddc0af81fed0f125bd18e9799a065b014ab166]",
				},
				output1: nil,
				output2: &KeepArchive{
					Name:   "down-2024-07-21-16-21-11",
					Reason: "daily[oldest] #3",
				},
			},
			{
				input: LogMessage{
					Message: "terminating with success status, rc 0",
				},
				output1: nil,
				output2: nil,
			},
		}

		// ACT
		for _, test := range inputOutput {
			output1, output2 := parsePruneOutput(test.input)

			// ASSERT
			Expect(output1).To(Equal(test.output1))
			Expect(output2).To(Equal(test.output2))
		}
	})
})
