RELAY FORM — BOLD
Version 1.000

A custom geometric bold font reconstructed and extended solely from the
supplied five-letter "Relay" image. R, e, l, a and y follow its outlines and
spacing; the remaining characters are new designs based on those shapes.
This is a reconstruction and extension, not an extraction of the original font.
No existing font was used as a source for the glyph outlines.

FILES
  fonts/RelayForm-Bold.ttf    TrueType desktop font, with automatic hinting
  fonts/RelayForm-Bold.otf    OpenType desktop font with genuine CFF outlines
  fonts/RelayForm-Bold.woff   Compressed web font
  fonts/RelayForm-Bold.woff2  More compact web font
  relay-form.css             Ready-to-use web font declaration and classes
  Preview.html               Editable browser preview; open after extracting
  Specimen.png               High-resolution specimen rendered from the font
  Character-coverage.txt     Complete list of supported Unicode characters
  Validation.txt             Results of format, rendering and shaping checks

PREMIERE / WINDOWS
  1. Extract the ZIP first.
  2. Right-click RelayForm-Bold.ttf and select Install (or Install for all users).
     You can also open the file and use the Install button.
  3. Restart Premiere if it was already open.
  4. Select your text and choose the family "Relay Form", style "Bold".

  OTF is an alternative desktop format. Install either the TTF or the OTF;
  both have the same family and style names, so there is no benefit to
  installing both. WOFF and WOFF2 are for websites.

MACOS
  Open either desktop font in Font Book, install it, then restart your editor
  if needed. Select Relay Form, Bold.

WEB
  Keep relay-form.css next to the fonts folder. Link it from your page:

    <link rel="stylesheet" href="/assets/relay-form.css">

  Then apply the supplied class:

    <h1 class="relay-form">Make your next move.</h1>

  Or set these properties on your own selector:

    font-family: "Relay Form", Arial, sans-serif;
    font-weight: 700;
    font-style: normal;
    font-kerning: normal;
    line-height: 1.28;

  For equal-width digits, add font-variant-numeric: tabular-nums;
  For very tight headings, adjust letter-spacing cautiously. The reference
  word already has its own kerning, so start at letter-spacing: 0.
  Use extra line spacing for text with accents below descending letters.
  If fonts are served from a different domain, configure that server's CORS
  headers to allow your site to load them.

CHARACTERS AND FEATURES
  One upright weight: Bold / 700.
  525 encoded Unicode characters, including spaces and combining marks.
  537 glyphs, including ten tabular-digit alternates.
  Complete printable ASCII, Latin-1 Supplement and Latin Extended-A.
  Additional Latin accents, smart quotes, dashes, currency and math symbols.
  GPOS kerning, legacy Latin-1 kerning in TTF, combining-mark placement,
  and the OpenType tnum feature for tabular digits.
  Font embedding restrictions are not set (OS/2 fsType = 0).

  The design is intended for headlines, titles, branding and large captions.
  It does not include Arabic, Cyrillic, Greek, CJK or an italic/regular weight.
  See Character-coverage.txt for the exact character list.

VALIDATION
  See Validation.txt for the checks actually performed.
  Premiere itself was not used to test this package.
