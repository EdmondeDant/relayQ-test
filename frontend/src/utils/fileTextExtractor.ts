import JSZip from 'jszip'
import mammoth from 'mammoth/mammoth.browser'
import * as pdfjsLib from 'pdfjs-dist'
import { createWorker } from 'tesseract.js'
import * as XLSX from 'xlsx'

pdfjsLib.GlobalWorkerOptions.workerSrc = new URL('pdfjs-dist/build/pdf.worker.mjs', import.meta.url).toString()

export interface ExtractedFileText {
  fileName: string
  text: string
}

const MAX_TEXT_LENGTH_PER_FILE = 60_000

function trimText(text: string): string {
  const normalized = text.replace(/\r\n/g, '\n').replace(/\n{3,}/g, '\n\n').trim()
  return normalized.length > MAX_TEXT_LENGTH_PER_FILE
    ? `${normalized.slice(0, MAX_TEXT_LENGTH_PER_FILE)}\n\n[内容过长，已截断]`
    : normalized
}

function extensionOf(file: File): string {
  return file.name.split('.').pop()?.toLowerCase() || ''
}

function isTextLike(file: File): boolean {
  return file.type.startsWith('text/') || ['txt', 'md', 'csv', 'json', 'log', 'xml', 'html'].includes(extensionOf(file))
}

async function extractTextFile(file: File): Promise<string> {
  return trimText(await file.text())
}

async function extractPdf(file: File): Promise<string> {
  const buffer = await file.arrayBuffer()
  const pdf = await pdfjsLib.getDocument({ data: buffer }).promise
  const pages: string[] = []

  for (let pageNumber = 1; pageNumber <= pdf.numPages; pageNumber += 1) {
    const page = await pdf.getPage(pageNumber)
    const content = await page.getTextContent()
    const text = content.items.map((item) => ('str' in item ? item.str : '')).join(' ')
    pages.push(`第 ${pageNumber} 页\n${text}`)
  }

  return trimText(pages.join('\n\n'))
}

async function extractWord(file: File): Promise<string> {
  if (extensionOf(file) !== 'docx') {
    throw new Error('暂只支持 .docx 格式，旧版 .doc 请先另存为 .docx')
  }
  const result = await mammoth.extractRawText({ arrayBuffer: await file.arrayBuffer() })
  return trimText(result.value)
}

async function extractExcel(file: File): Promise<string> {
  const workbook = XLSX.read(await file.arrayBuffer(), { type: 'array' })
  const sheets = workbook.SheetNames.map((sheetName) => {
    const sheet = workbook.Sheets[sheetName]
    const csv = XLSX.utils.sheet_to_csv(sheet)
    return `工作表：${sheetName}\n${csv}`
  })
  return trimText(sheets.join('\n\n'))
}

async function extractPowerPoint(file: File): Promise<string> {
  if (extensionOf(file) !== 'pptx') {
    throw new Error('暂只支持 .pptx 格式，旧版 .ppt 请先另存为 .pptx')
  }

  const zip = await JSZip.loadAsync(await file.arrayBuffer())
  const slideFiles = Object.keys(zip.files)
    .filter((name) => /^ppt\/slides\/slide\d+\.xml$/.test(name))
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))

  const slides: string[] = []
  const parser = new DOMParser()

  for (const [index, name] of slideFiles.entries()) {
    const xml = await zip.files[name].async('text')
    const doc = parser.parseFromString(xml, 'application/xml')
    const texts = Array.from(doc.getElementsByTagName('a:t')).map((node) => node.textContent || '')
    slides.push(`第 ${index + 1} 页\n${texts.join('\n')}`)
  }

  return trimText(slides.join('\n\n'))
}

async function extractImage(file: File): Promise<string> {
  const worker = await createWorker('chi_sim+eng')
  try {
    const result = await worker.recognize(file)
    return trimText(result.data.text)
  } finally {
    await worker.terminate()
  }
}

export async function extractTextFromFile(file: File): Promise<ExtractedFileText> {
  const ext = extensionOf(file)
  let text = ''

  if (isTextLike(file)) {
    text = await extractTextFile(file)
  } else if (ext === 'pdf') {
    text = await extractPdf(file)
  } else if (['doc', 'docx'].includes(ext)) {
    text = await extractWord(file)
  } else if (['xls', 'xlsx'].includes(ext)) {
    text = await extractExcel(file)
  } else if (['ppt', 'pptx'].includes(ext)) {
    text = await extractPowerPoint(file)
  } else if (file.type.startsWith('image/')) {
    text = await extractImage(file)
  } else {
    throw new Error(`不支持的文件类型：${file.name}`)
  }

  return {
    fileName: file.name,
    text: text || '[未提取到文字内容]',
  }
}

export async function extractTextFromFiles(files: File[]): Promise<ExtractedFileText[]> {
  const results: ExtractedFileText[] = []
  for (const file of files) {
    results.push(await extractTextFromFile(file))
  }
  return results
}
