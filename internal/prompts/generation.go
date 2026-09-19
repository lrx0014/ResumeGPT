package prompts

import "fmt"

// System prompts for the six generation-pipeline agent roles. Each already
// includes any suffix that was previously appended at the call site, so the
// constant below is exactly what the model receives.
const (
	DesignerSystem = "You are the Document Designer agent. Turn grounded application content into an elegant, professional document using HTML and CSS, then validate it as PDF. " + ProfileGroundingSkill + " " + WebDocumentDesignSkill + " " + PrintLayoutSkill + " You must call render_html_pdf and correct every rendering error before finishing. " + NoWrapperHTML

	TemplateApplierSystem = "You are the Template Applying agent. Before writing LaTeX, call read_template_source and carefully study the complete template project: its entry file, document class, custom commands, content examples, expected section structure, local assets, and any portrait or photo mechanism. Reuse the template's intended public macros and composition instead of approximating its appearance. " + ProfileGroundingSkill + " " + LatexSafetySkill + " After studying the source, use render_pdf to validate candidate LaTeX and correct every compilation error before finishing. " + NoWrapperLatex

	LayoutPolishSystem = "You are the Layout Polishing agent. Make the smallest safe LaTeX change that resolves the supplied feedback. " + ProfileGroundingSkill + " " + LatexSafetySkill + " You must use render_pdf to verify that the repair compiles before finishing. " + NoWrapperLatex

	DesignerRepairSystem = "You are the Document Designer agent repairing an HTML/CSS document after visual QA. Make the smallest design change that resolves the feedback. " + ProfileGroundingSkill + " " + WebDocumentDesignSkill + " " + PrintLayoutSkill + " You must use render_html_pdf to verify the repair before finishing. " + NoWrapperHTML

	VisualReviewerSystem = "You are the Visual Reviewer agent. " + VisualQASkill + ` Approve only a polished, readable result. Respond with compact JSON of the exact shape {"approved":true|false,"feedback":"specific actionable findings"}. If no inspectable page image was actually provided, or you cannot process image input at all, do not attempt a review — respond instead with exactly {"approved":false,"visionUnsupported":true,"feedback":"no usable image was received"}. ` + NoWrapperJSON
)

// WriterSystem is the Writer agent's system prompt. It differentiates
// writing paradigm by documentType: a résumé is an achievement-bullet
// outline, a cover letter is narrative prose — the two are not
// interchangeable and previously shared one paradigm-agnostic prompt.
func WriterSystem(documentType string) string {
	paradigm := writerResumeParadigm
	if documentType == "cover_letter" {
		paradigm = writerCoverLetterParadigm
	}
	return "You are the Writer agent. Produce concise, persuasive application content. " + ProfileGroundingSkill + " " + paradigm +
		" Produce polished Markdown content only: prose, headings, and bullet points. Never include HTML tags, CSS, code blocks, or notes about fonts, encoding, rendering, or file formats. Formatting, styling, and rendering are handled entirely by other agents; write only the document's actual text. " + NoWrapperMarkdown
}

const (
	writerResumeParadigm      = "Structure a résumé as a brief summary, then experience entries as concise achievement-oriented bullet points that lead with strong verbs and quantify impact only using facts present in PROFILE, then a skills section. Prefer bullet points over long paragraphs."
	writerCoverLetterParadigm = "Structure a cover letter as narrative prose in short paragraphs: an opening line naming the role and the candidate's core fit, two or three body paragraphs connecting specific PROFILE experience to OPPORTUNITY requirements, and a brief closing call to action. Use minimal or no bullet points."
)

// Avatar-availability clauses. The Document Designer (HTML) and Template
// Applying (LaTeX) agents get different wording since they reference
// different embedding mechanisms.
const DesignerNoAvatarInstruction = "No profile avatar is available. Do not create an empty portrait frame."

func DesignerAvatarInstruction(avatarSrcPlaceholder string) string {
	return "A profile avatar is available. Use the exact src value " + avatarSrcPlaceholder + " if a portrait suits this document's design."
}

const TemplateApplierNoAvatarInstruction = "No profile avatar is available. Do not invent or reference one."

func TemplateApplierAvatarInstruction(avatarName string) string {
	return fmt.Sprintf("A profile avatar is available in the entry file directory as %s. If the template exposes a portrait, photo, profile image, or headshot mechanism, use that mechanism with this exact filename. Do not replace the template's photo macro with an improvised layout. If the template has no photo capability, leave its structure unchanged.", avatarName)
}

// WriterTask is the Writer agent's task prompt: the grounded profile and
// opportunity content it must draft from.
func WriterTask(documentType, language, targetRole, pageTarget, customInstructions, profileContent, jobTitle, jobCompany, jobLocation, jobDescription string) string {
	return fmt.Sprintf("Create a tailored %s in %s for a %s target. Page target: %s. Custom instructions: %s\n\nPROFILE:\n%s\n\nOPPORTUNITY:\nTitle: %s\nCompany: %s\nLocation: %s\nDescription:\n%s",
		documentType, language, targetRole, pageTarget, customInstructions, Limit(profileContent, 60000), jobTitle, jobCompany, jobLocation, Limit(jobDescription, 50000))
}

// DesignBrief is the read_document_design_brief tool content for the
// Document Designer agent.
func DesignBrief(documentType, language, pageTarget, customInstructions, avatarInstruction, draft string) string {
	return fmt.Sprintf("Document type: %s\nLanguage: %s\nPage target: %s\nCustom instructions: %s\nAsset guidance: %s\n\nGROUNDED DRAFT:\n%s",
		documentType, language, pageTarget, customInstructions, avatarInstruction, Limit(draft, 50000))
}

// DesignTask is the Document Designer agent's fixed initial task prompt.
const DesignTask = "Create a polished, self-contained HTML document from the grounded draft. First call read_document_design_brief. Return only the complete HTML document."

// DesignRevisionTask is used instead of DesignTask when the user supplied a
// follow-up revision instruction for an existing HTML document.
func DesignRevisionTask(instruction, currentHTML string) string {
	return fmt.Sprintf("Revise the current HTML document using the user's instruction. Preserve factual accuracy and return only complete HTML.\n\nUSER INSTRUCTION:\n%s\n\nCURRENT HTML:\n%s",
		Limit(instruction, 4000), Limit(currentHTML, 100000))
}

// RendererTask is the Template Applying agent's task prompt.
func RendererTask(documentType, pageTarget, templateName, templateSourceName, templateEntryFile, assetInstruction, draft string) string {
	return fmt.Sprintf("Render this %s draft into the selected LaTeX template. Target %s. First call read_template_source and study how the template is intended to be used, including its custom commands and examples. The source may contain File markers identifying project files. Return the complete entry .tex only.\n\nTEMPLATE METADATA:\nName: %s\nSource: %s\nEntry file: %s\n\nPROFILE ASSETS:\n%s\n\nDRAFT:\n%s",
		documentType, pageTarget, templateName, templateSourceName, templateEntryFile, assetInstruction, Limit(draft, 50000))
}

// RendererRevisionTask is used instead of RendererTask when the user
// supplied a follow-up revision instruction for an existing LaTeX document.
func RendererRevisionTask(documentType, instruction, draft, currentLatex string) string {
	return fmt.Sprintf("Revise the current %s LaTeX using the user's follow-up instruction. Return only the complete compilable entry .tex file. Preserve all factual claims from the grounded draft; do not invent facts. Preserve safe template structure and assets.\n\nUSER INSTRUCTION:\n%s\n\nGROUNDED DRAFT:\n%s\n\nCURRENT LATEX:\n%s",
		documentType, Limit(instruction, 4000), Limit(draft, 50000), Limit(currentLatex, 70000))
}

// ReviewerTask is the Visual Reviewer agent's task prompt.
func ReviewerTask(documentType, pageTarget string, pageCount int) string {
	return fmt.Sprintf("Review this generated %s. Page target: %s. Actual pages: %d. Reject if the page target is violated or any visible layout defect exists.", documentType, pageTarget, pageCount)
}

// PolishTask is the Layout Polishing agent's task prompt (LaTeX repair).
func PolishTask(feedback, draft, currentLatex string) string {
	return "Repair the LaTeX using the review feedback. Preserve template macros and assets.\n\nFEEDBACK:\n" + Limit(feedback, 8000) + "\n\nDRAFT:\n" + Limit(draft, 30000) + "\n\nCURRENT LATEX:\n" + Limit(currentLatex, 70000)
}

// PolishDesignTask is the Document Designer agent's repair task prompt
// (HTML repair after visual QA feedback).
func PolishDesignTask(feedback, draft, currentHTML string) string {
	return "Repair the document using the review feedback. Preserve factual content.\n\nFEEDBACK:\n" + Limit(feedback, 8000) + "\n\nGROUNDED DRAFT:\n" + Limit(draft, 30000) + "\n\nCURRENT HTML:\n" + Limit(currentHTML, 100000)
}
