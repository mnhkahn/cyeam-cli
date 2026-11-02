// Native OCR only; never consult calculated answers or replace digit glyphs.
ObjC.import('Foundation');
ObjC.import('Vision');
ObjC.import('CoreImage');
ObjC.import('AppKit');
function rect(x,y,w,h) { return {origin:{x:x,y:y},size:{width:w,height:h}}; }
function affine(a,b,c,d,x,y) { return {a:a,b:b,c:c,d:d,tx:x,ty:y}; }
function run() {
    var data = $.NSFileHandle.fileHandleWithStandardInput.readDataToEndOfFile;
    var image = $.CIImage.imageWithData(data);
    var extent = image.extent, width = extent.size.width, height = extent.size.height;
    function recognize(input, local, correction) {
        var r = $.VNRecognizeTextRequest.alloc.init;
        r.recognitionLevel = 0; // accurate
        r.recognitionLanguages = $(local ? ['en-US'] : ['zh-Hans', 'en-US']);
        r.usesLanguageCorrection = correction;
        r.minimumTextHeight = 0;
        if (correction) r.customWords = $(['0','1','2','3','4','5','6','7','8','9']);
        // Materialize the finite crop; Vision's CIImage handler can otherwise
        // inspect the uncropped filter graph and return no observations.
        var handler;
        if (local) {
            var pixels = $.NSBitmapImageRep.alloc.initWithCIImage(input);
            var png = pixels.representationUsingTypeProperties(4, $.NSDictionary.dictionary);
            handler = $.VNImageRequestHandler.alloc.initWithDataOptions(png, $.NSDictionary.dictionary);
        } else {
            handler = $.VNImageRequestHandler.alloc.initWithCIImageOptions(input, $.NSDictionary.dictionary);
        }
        var error = Ref();
        if (!handler.performRequestsError($.NSArray.arrayWithObject(r), error)) throw Error('Vision failed');
        var out = [];
        for (var i = 0; i < r.results.count; ++i) {
            var ob = r.results.objectAtIndex(i), candidates = ob.topCandidates(1);
            if (candidates.count) out.push({ob: ob, candidate: candidates.objectAtIndex(0)});
        }
        return out;
    }
    function box(rect) {
        return [rect.origin.x * width, (1-rect.origin.y-rect.size.height)*height,
                (rect.origin.x+rect.size.width)*width, (1-rect.origin.y)*height];
    }
    function textLine(text, b, confidence) { return {text:text, box:b, confidence:confidence, page:1}; }
    function processed(crop, wide) {
        var im = crop;
        if (wide) im = im.imageByApplyingFilterWithInputParameters('CIColorControls',
            $({inputSaturation:0, inputContrast:1.6, inputBrightness:0.10}));
        im = im.imageByApplyingTransform(affine(wide ? 3 : 2, 0, 0, 2, 0, 0));
        if (!wide) im = im.imageByApplyingTransform(affine(1,0,-0.25,1,0,0));
        var e = im.extent;
        im = im.imageByApplyingTransform(affine(1,0,0,1,40-e.origin.x,40-e.origin.y));
        var canvas = rect(0,0,Math.ceil(e.size.width)+80,Math.ceil(e.size.height)+80);
        var white = $.CIImage.imageWithColor($.CIColor.colorWithRedGreenBlueAlpha(1,1,1,1)).imageByCroppingToRect(canvas);
        return im.imageByCompositingOverImage(white).imageByCroppingToRect(canvas);
    }
    var original = recognize(image, false, false), lines = [], rawLines = [], alternatives = [];
    for (var i = 0; i < original.length; ++i) {
        var candidate = original[i].candidate, text = ObjC.unwrap(candidate.string);
        var b = box(original[i].ob.boundingBox), confidence = candidate.confidence;
        rawLines.push(textLine(text,b,confidence));
        // Separate adjacent numbered questions merged into a single observation.
        var labels = [], labelRE = /[（(]\d+[）)]\s*\d+\s*[×xX+÷/]/g, match;
        while ((match = labelRE.exec(text)) !== null) labels.push(match.index);
        if (!labels.length || labels[0] !== 0) labels.unshift(0);
        for (var j = 0; j < labels.length; ++j) {
            var start = labels[j], end = j+1 < labels.length ? labels[j+1] : text.length;
            var fragment = text.substring(start,end), fragmentBox = b;
            try {
                fragmentBox = box(candidate.boundingBoxForRangeError($.NSMakeRange(start,end-start),Ref()).boundingBox);
            } catch (_) {}
            // A dash after a complete multiplication/division stem is an OCR
            // separator ambiguity, not a subtraction operation. Raw text is kept.
            var normalized = fragment.replace(/^([\s]*[（(]\d+[）)]\s*\d+\s*[×xX÷/]\s*\d+)\s*[-–]\s*/, '$1=');
            lines.push(textLine(normalized,fragmentBox,confidence));
            var eq = normalized.indexOf('=');
            if (eq < 0) continue;
            var answer = normalized.substring(eq+1).trim();
            // Only retry malformed answer regions. No retry triggered by a
            // comparison to a computed value, and no low-ranked candidate search.
            if (/^[+-]?(?:\d+(?:\.\d+)?|\.\d+)$/.test(answer)) continue;
            var originalEq = fragment.indexOf('=');
            if (originalEq < 0) continue;
            var answerRange = $.NSMakeRange(start+originalEq+1,end-start-originalEq-1);
            try {
                var rb = candidate.boundingBoxForRangeError(answerRange,Ref()).boundingBox;
                var x = Math.max(0, Math.floor(rb.origin.x*width)-18);
                var y = Math.max(0, Math.floor(rb.origin.y*height)-18);
                var right = Math.min(width,Math.ceil((rb.origin.x+rb.size.width)*width)+18);
                var top = Math.min(height,Math.ceil((rb.origin.y+rb.size.height)*height)+18);
                if (right <= x || top <= y) continue;
                var crop = image.imageByCroppingToRect(rect(x,y,right-x,top-y));
                for (var mode = 0; mode < 2; ++mode) {
                    var local = recognize(processed(crop, mode === 0),true,mode === 0);
                    // More than one text observation is ambiguous; retain only
                    // a single complete first reading, never concatenate digits.
                    if (local.length !== 1) continue;
                    var value = ObjC.unwrap(local[0].candidate.string);
                    alternatives.push({kind:mode === 0 ? 'wide-contrast' : 'deskew',
                        line:textLine(normalized.substring(0,eq+1)+value,fragmentBox,local[0].candidate.confidence)});
                }
            } catch (_) { /* Keep the original if a crop cannot be recognized. */ }
        }
    }
    return JSON.stringify({lines:lines, raw_lines:rawLines, alternatives:alternatives});
}
