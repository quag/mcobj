package main
import (
"io"
"regexp"
"encoding/line"
"strconv"
"fmt"
"strings"
)

type ObjVec2 struct {
  x float32;
  y float32;
}
type ObjVec3 struct {
  x float32;
  y float32;
  z float32;
}
type ObjReference struct {
	vertex int;
	texcoord int;//-1 for missing
	normal int;// -1 for missing
}
type ObjFile struct{
  texcoords []ObjVec2;
  vertices []ObjVec3;
  normals []ObjVec3;
  faces [][]ObjReference;
  
}

func parseObj(f io.Reader) (retval ObjFile) {
    //integerNumber:="-{0,1}[0-9]+";
    polygonVertex:="(-{0,1}[0-9]+)(/-{0,1}[0-9]*)?(/-{0,1}[0-9]*)?"
    //materialLibRE:=regexp.MustCompile("^mtllib[ \t]+([^#]*)");
    floatingPointNumber:="([-+]?[0-9]*\\.?[0-9]+([eE][-+]?[0-9]+)?)";
    //usemtlRE:=regexp.MustCompile("^usemtl[ \t]+([^#]*)");
    vertexRE:=regexp.MustCompile("^v[ \t]+"+floatingPointNumber+"[ \t]+"+floatingPointNumber+"[ \t]+"+floatingPointNumber);
    normalRE:=regexp.MustCompile("^vn[ \t]+"+floatingPointNumber+"[ \t]+"+floatingPointNumber+"[ \t]+"+floatingPointNumber);
    texCoordRE:=regexp.MustCompile("^vt[ \t]+"+floatingPointNumber+"[ \t]+"+floatingPointNumber);
    polygonRE:=regexp.MustCompile("^f[ \t]+("+polygonVertex+"([ \t]+"+polygonVertex+")*"+")");
    reader :=line.NewReader(f,16384);
    for linenum:=0;true;linenum++ {
        line,_,err:=reader.ReadLine()
        if err!=nil {//EOF?
            break;
        }
        vertex:=vertexRE.FindSubmatch(line)
        normal:=normalRE.FindSubmatch(line)
        texCoord:=texCoordRE.FindSubmatch(line)
        polygon:=polygonRE.FindSubmatch(line)
        if (vertex!=nil) {
			x,err0:=strconv.Atof32(string(vertex[1]))
			y,err1:=strconv.Atof32(string(vertex[2]))
			z,err2:=strconv.Atof32(string(vertex[3]))
			if (err0!=nil||err1!=nil||err2!=nil) {
				fmt.Println("Error parsing obj vertex file, line: ",linenum,err0,err1,err2);
			}
            retval.vertices=append(retval.vertices,ObjVec3{x,y,z});
        }
        if (normal!=nil) {
			x,err0:=strconv.Atof32(string(normal[1]))
			y,err1:=strconv.Atof32(string(normal[2]))
			z,err2:=strconv.Atof32(string(normal[3]))
			if (err0!=nil||err1!=nil||err2!=nil) {
				fmt.Println("Error parsing obj normal file, line: ",linenum,err0,err1,err2);
			}
            retval.normals=append(retval.normals,ObjVec3{x,y,z});
        }
        if (texCoord!=nil) {
			u,err0:=strconv.Atof32(string(texCoord[1]))
			v,err1:=strconv.Atof32(string(texCoord[2]))
			if (err0!=nil||err1!=nil) {
				fmt.Println("Error parsing obj texcoord file, line: ",linenum,err0,err1);
			}
            retval.texcoords=append(retval.texcoords,ObjVec2{u,v});              
        }
        if (polygon!=nil) {
			faces := strings.Split(strings.Replace(string(polygon[1]),"\t"," ",-1)," ",-1);
			face := make ([]ObjReference,0);
			for _,facestr := range faces {
				references:=strings.Split(facestr,"/",3);
				if (len(references)!=0) {
					v,errv:=strconv.Atoi(references[0])
					if (errv==nil) {
						curFaceVertex:=ObjReference{-1,-1,-1};
						if (v<0){
							curFaceVertex.vertex=len(retval.vertices)+v;
						}else {
							curFaceVertex.vertex=v;
						}
						if (len(references)>1) {
							t,errt:=strconv.Atoi(references[1])
							if (errt==nil) {
								if (t<0){
									curFaceVertex.texcoord=len(retval.texcoords)+t;
								}else {
									curFaceVertex.texcoord=t;
								}
							}
						}
						if (len(references)>2) {
							n,errn:=strconv.Atoi(references[2])
							if (errn==nil) {
								if (n<0){
									curFaceVertex.normal=len(retval.normals)+n;
								}else {
									curFaceVertex.normal=n;
								}
							}
						}
						face=append(face,curFaceVertex);              
					}
				}
            }
			if (len(face)>2) {
				retval.faces=append(retval.faces,face);
			}
        }
    }
	return;
}