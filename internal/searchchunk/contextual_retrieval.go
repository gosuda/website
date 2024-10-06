package searchchunk

const prompt = `You will be provided with a document and a chunk of text from that document.
Your goal is to provide a short, succinct context to situate the chunk within the overall document.
Your response must be formatted with a start token and end token.

Here is the document:
<document>
{$WHOLE_DOCUMENT}
</document>

Here is the chunk of text:
<chunk>
{$CHUNK_CONTENT}
</chunk>

First, locate the chunk within the document.

Then, provide a short, succinct context to situate this chunk within the overall document.
The context should help someone searching for the chunk understand where it appears in the document.

Begin your response with "[START_TOKEN]" and end it with "[END_TOKEN]".
Only the context should appear between the tokens. Do not add any additional commentary or explanation.

For example:

[START_TOKEN]
...
[END_TOKEN]`
