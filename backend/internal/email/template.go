package email

import (
	"fmt"
	"html"
	"strings"
)

type emailLayoutData struct {
	Preheader   string
	Eyebrow     string
	Heading     string
	Greeting    string
	Intro       string
	ButtonLabel string
	Link        string
	Highlight   string
	Footnote    string
}

// renderHTML menyusun surel HTML monokrom bergaya rekavia.com yang kompatibel
// dengan Gmail (desktop/mobile), Apple Mail, dan Outlook. Semua input dinamis
// di-escape lewat html.EscapeString agar aman dari injeksi HTML.
func renderHTML(data emailLayoutData) string {
	baseURL := strings.TrimRight(strings.TrimSpace(PublicBaseURL), "/")
	logoURL := ""
	if strings.HasPrefix(baseURL, "https://") || strings.HasPrefix(baseURL, "http://") {
		logoURL = baseURL + "/brand/logo-dark-64.png"
	}

	preheader := html.EscapeString(data.Preheader)
	eyebrow := html.EscapeString(data.Eyebrow)
	heading := html.EscapeString(data.Heading)
	greeting := html.EscapeString(data.Greeting)
	intro := html.EscapeString(data.Intro)
	buttonLabel := html.EscapeString(data.ButtonLabel)
	safeLink := html.EscapeString(data.Link)
	highlight := html.EscapeString(data.Highlight)
	footnote := html.EscapeString(data.Footnote)

	logoBlock := ""
	if logoURL != "" {
		logoBlock = fmt.Sprintf(
			`<img src="%s" alt="ILA" width="24" height="33" style="display:block;border:0;height:28px;width:auto;" />`,
			html.EscapeString(logoURL),
		)
	}

	highlightBlock := ""
	if highlight != "" {
		highlightBlock = fmt.Sprintf(`
                <tr>
                  <td style="padding:18px 0 0 0;">
                    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;background-color:#f5f5f7;border:1px solid #e4e4e7;border-radius:10px;">
                      <tr>
                        <td style="padding:12px 16px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:12px;line-height:1.55;color:#3f3f46;">
                          %s
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>`, highlight)
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="id">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1.0" />
    <meta name="color-scheme" content="light" />
    <title>%s</title>
  </head>
  <body style="margin:0;padding:0;background-color:#f5f5f7;-webkit-font-smoothing:antialiased;">
    <div style="display:none;max-height:0;overflow:hidden;opacity:0;mso-hide:all;">%s</div>
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;background-color:#f5f5f7;">
      <tr>
        <td align="center" style="padding:32px 16px;">
          <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:560px;border-collapse:collapse;background-color:#ffffff;border:1px solid #e4e4e7;border-radius:16px;overflow:hidden;">
            <!-- Header Obsidian -->
            <tr>
              <td style="background-color:#050507;padding:22px 28px;">
                <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;">
                  <tr>
                    <td style="vertical-align:middle;width:36px;">%s</td>
                    <td style="vertical-align:middle;padding-left:10px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
                      <div style="font-size:14px;font-weight:600;letter-spacing:-0.01em;color:#ffffff;line-height:1.2;">Imaginary Liquidity</div>
                      <div style="font-size:11px;color:#a1a1aa;line-height:1.3;margin-top:2px;">Analisis Likuiditas &amp; Eksekusi Presisi</div>
                    </td>
                  </tr>
                </table>
              </td>
            </tr>

            <!-- Isi Utama -->
            <tr>
              <td style="padding:30px 28px 24px 28px;">
                <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;">
                  <tr>
                    <td style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:11px;font-weight:600;letter-spacing:0.14em;text-transform:uppercase;color:#71717a;">
                      %s
                    </td>
                  </tr>
                  <tr>
                    <td style="padding-top:8px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:22px;font-weight:600;letter-spacing:-0.02em;line-height:1.25;color:#09090b;">
                      %s
                    </td>
                  </tr>
                  <tr>
                    <td style="padding-top:18px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:14px;line-height:1.65;color:#18181b;">
                      %s
                    </td>
                  </tr>
                  <tr>
                    <td style="padding-top:10px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:14px;line-height:1.65;color:#52525b;">
                      %s
                    </td>
                  </tr>

                  <!-- Tombol Aksi -->
                  <tr>
                    <td style="padding-top:24px;">
                      <a href="%s" target="_blank" rel="noopener noreferrer" style="display:inline-block;background-color:#09090b;color:#ffffff;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:14px;font-weight:600;line-height:1;text-decoration:none;padding:14px 24px;border-radius:10px;">
                        %s
                      </a>
                    </td>
                  </tr>
                  %s
                  <!-- Tautan cadangan -->
                  <tr>
                    <td style="padding-top:22px;border-top:1px solid #f4f4f5;margin-top:22px;">
                      <div style="padding-top:16px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:12px;line-height:1.5;color:#71717a;">
                        Jika tombol di atas tidak terbuka, salin tautan berikut ke peramban Anda:
                      </div>
                      <div style="margin-top:6px;padding:10px 12px;background-color:#fafafa;border:1px solid #e4e4e7;border-radius:8px;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;font-size:11px;line-height:1.5;color:#27272a;word-break:break-all;">
                        <a href="%s" target="_blank" rel="noopener noreferrer" style="color:#09090b;text-decoration:none;">%s</a>
                      </div>
                    </td>
                  </tr>
                </table>
              </td>
            </tr>

            <!-- Footer -->
            <tr>
              <td style="background-color:#fafafa;border-top:1px solid #f4f4f5;padding:18px 28px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:11px;line-height:1.6;color:#71717a;">
                <div>%s</div>
                <div style="margin-top:6px;color:#a1a1aa;">Imaginary Liquidity · rekavia.com</div>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>`,
		heading,
		preheader,
		logoBlock,
		eyebrow,
		heading,
		greeting,
		intro,
		safeLink,
		buttonLabel,
		highlightBlock,
		safeLink,
		safeLink,
		footnote,
	)
}
