mcobj: Minecraft to OBJ (and PRT) converter
===========================================

![](http://github.com/downloads/quag/mcobj/splash1.jpg)
![](http://github.com/downloads/quag/mcobj/splash2.jpg)

For example renders see [http://quag.imgur.com/minecraft__blender](http://quag.imgur.com/minecraft__blender).

If you'd like to show me renders, report problems, suggest improvements, would like builds for other platforms or to ask for help, please email me.

The [http://reddit.com/r/mcobj](r/mcobj) sub-reddit been setup for showing off renders, discussing how to achieve nice effects and news about updates. Feel free to contribute and if it is quiet or empty to stir it up with some posts.

As I'd love to see renders, please email me copies of images you create.

Usage
-----

On Windows:

    mcobj -cpu 4 -s 20 -o world1.obj %AppData%\.minecraft\saves\World1

On OSX:

    mcobj -cpu 4 -s 20 -o world1.obj ~/Library/Application\ Support/minecraft/saves/World1

On Linux:

    mcobj -cpu 4 -s 20 -o world1.obj ~/.minecraft/saves/World1

Flags:

<table>
      <tbody><tr><td>-cpu 4</td><td>How many cores to use while processing. Defaults to 1. Set to the number of cpu's in the machine.</td></tr>
      <tr><td>-o a.obj</td><td>Name for the obj file to write to. Defaults to a.obj</td></tr>
      <tr><td>-h</td><td>Help</td></tr>
      <tr><td>-prt</td><td>Output a <a href="http://software.primefocusworld.com/software/support/krakatoa/prt_file_format.php">PRT</a> file instead of OBJ</td></tr>
      <tr><td>-3dsmax=false</td><td>Output an obj file that is incompatible with 3dsMax. Typically is faster, uses less memory and results in a smaller .obj files</td></tr>
    </tbody></table>

Chunk Selection:

<table>
      <tbody>
      <tr><td>-x -8.4 -z 272.8</td><td>Center the output to chunk x=-1 and z=17. Defaults to chunk 0,0</td></tr>
      <tr><td>-cx 10 -cz -23</td><td>Center the output to chunk x=10 and z=23. Defaults to chunk 0,0. To calculate the chunk coords, divide the values given in Minecraft's F3 screen by 16</td></tr>
      <tr><td>-s 20</td><td>Output a sized square of chunks centered on -cx -cz. -s 20 will output 20x20 area around 0,0</td></tr>
      <tr><td>-rx 2 -rx 8</td><td>Output a sized rectangle of chunks centered on -cx -cz. -rx 2 -rx 8 will output a 2x8 area around 0,0</td></tr>
    </tbody></table>

Limit the output:

<table>
      <tbody><tr><td>-fk 300</td><td>Limit the face count (in thousands of faces)</td></tr>
      <tr><td>-y 63</td><td>Omit all blocks below this height. Use 63 for sea level</td></tr>
      <tr><td>-hb</td><td>Hide the bottom of the world</td></tr>
      <tr><td>-g</td><td>Gray; omit materials</td></tr>
      <tr><td>-bf</td><td>Don't combine adjacent faces of the same block within a column</td></tr>
      <tr><td>-sides</td><td>Output sides of chunks at the edges of selection. Sides are usually omitted</td></tr>
    </tbody></table>

Change Log
---------

<table>
  <tr>
    <th></th>
    <th></th>
    <th>Change Log</th>
  </tr>
  <tr>
    <td>?</td>
    <td>0.13</td>
    <td>
      <ul>
        <li>Add blocks for 1.6</li>
        <li>Fix bug: EOF error on empty mcr files in beta worlds</li>
        <li>Fix bug: crash when -o refers to a missing directory</li>
        <li>Switch to goinstall based building (requires a weekly build newer than go-r57)</li>
        <li>PRT: invert x and z</li>
        <li>PRT: remove 2x scaling</li>
        <li>Generate .obj files that can be read without any special flags in 3dsmax</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-05-15</td>
    <td>0.12</td>
    <td>
      <ul>
        <li>Add -x and -z flags for centering using Minecraft's F3 coordinates</li>
        <li>Change to material names instead of numbers</li>
        <li>Update blocks.json to handle Minecraft 1.5 blocks</li>
        <li>Fix bug: materials with extra data, e.g., wool colors, were always grey</li>
        <li>Fix bug: some leaves showed up as grey blocks</li>
        <li>Fix bug: -s square not centered on -cx -cz (off-by-one)</li>
        <li>Add arrays of block data to blocks.json -- beds and saplings</li>
        <li>Reworked build and release scripts</li>
        <li>Update source to compile with go release.r57.1</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-04-17</td>
    <td>0.11</td>
    <td>
      <ul>
        <li>Add -solid flag for putting sides on the area</li>
        <li>Add -rx 2 -rz 8 for selecting a rectangular area</li>
        <li>Slight improvement to glass color (less likely to be black)</li>
        <li>Read block data from blocks.json instead of being hard coded</li>
        <li>Remove -hs (hide stone) flag. Now add "empty":true to stone in blocks.json</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-03-17</td>
    <td>0.10.2</td>
    <td>
      <ul>
        <li>Add beds and redstone repeaters</li>
        <li>Update colors from WormSlayer</li>
        <li>Only invert x-axis for PRT (not both x and z)</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-03-14</td>
    <td>0.10.1</td>
    <td>
      <ul>
        <li>Bug fix: colors missing from blocks with extra data (except for wool) in OBJ</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-03-13</td>
    <td>0.10</td>
    <td>
      <ul>
        <li>Flip both x and z axis for PRT output</li>
        <li>Write out blocks around transparent blocks. For example, the block under a torch or the blocks around glass</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-03-06</td>
    <td>0.9.2</td>
    <td>
      <ul>
        <li>Rename PRT channel from MtlIndex to BlockId</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-03-06</td>
    <td>0.9.1</td>
    <td>
      <ul>
        <li>Fix bug (when PRT support was added) that made OBJ files empty</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-03-05</td>
    <td>0.9</td>
    <td>
      <ul>
        <li>Fix PRT export bug where all blocks appeared in a 16x16x128 space</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-03-05</td>
    <td>0.8</td>
    <td>
      <ul>
        <li>Add alternative output format: <a href="http://software.primefocusworld.com/software/support/krakatoa/prt_file_format.php">PRT</a> (use -prt flag to enable)</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-03-05</td>
    <td>0.7</td>
    <td>
      <ul>
        <li>Add support for the beta world format</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-02-19</td>
    <td>0.6</td>
    <td>
      <ul>
        <li>Further tweaks to colors</li>
        <li>Add color wool blocks</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-02-17</td>
    <td>0.5</td>
    <td>
      <ul>
        <li>Switch to WormSlayer's block colors</li>
        <li>Reverting to relative vertex numbers as abs numbers didn't help with the 3dsmax loading</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-02-17</td>
    <td>0.4</td>
    <td>
      <ul>
        <li>Use absolute vertex numbers instead of relative (to support 3dsmax)</li>
        <li>No longer leak file handles and gzip streams</li>
      </ul>
    </td>
  </tr>
  <tr>
    <td>2011-02-14</td>
    <td>0.3</td>
    <td>&nbsp;</td>
  </tr>
</table>

Limitations
-----------

Lots!

Only understands cube blocks (no torch meashes, door meshes and so on)
No textures. Just solid colors per block
Torches and lava don't emit light
...
The no-mesh and no-texture limitations are delibarate. They keep the obj file size and face counts down allowing dumps of large parts of the world without blowing out Blender's memory.

Blender
-------

Requires Blender 2.5.

Steps:

 1. Start Blender
 2. Delete the standard cube (right-click and press delete)
 3. Start the obj importer: (hit space, and type, "import obj" and hit return)
 4. Select the generated obj file and wait for it to import
 5. Enable Ambient Occlusion (switch to Blender's 'World' planel and click the 'Ambient Occlusion' checkbox)
 6. Start a render (hit F12)

Example Run
-----------

    C:\>mcobj -cpu 4 -s 4 -o world1.obj %AppData%\.minecraft\saves\World1
    mcobj dev-7-g1591dec (cpu: 4)
       1/5590 (  0,  0) Faces:  889 Size:  0.0MB
       2/5590 (  0, -1) Faces:  880 Size:  0.1MB
       3/5590 ( -1,  0) Faces:  850 Size:  0.1MB
       4/5590 ( -1, -1) Faces:  849 Size:  0.1MB
       5/5590 (  0,  1) Faces: 1042 Size:  0.2MB
       6/5590 ( -1,  1) Faces:  888 Size:  0.2MB
       7/5590 (  1,  0) Faces:  918 Size:  0.3MB
       8/5590 (  1, -1) Faces:  860 Size:  0.3MB
       9/5590 (  1,  1) Faces: 1678 Size:  0.4MB
      10/5590 (  0, -2) Faces:  906 Size:  0.4MB
      11/5590 ( -1, -2) Faces:  850 Size:  0.4MB
      12/5590 (  1, -2) Faces:  836 Size:  0.5MB
      13/5590 ( -2,  0) Faces:  863 Size:  0.5MB
      14/5590 ( -2, -1) Faces:  851 Size:  0.5MB
      15/5590 ( -2,  1) Faces:  909 Size:  0.6MB
      16/5590 ( -2, -2) Faces:  869 Size:  0.6MB

![](http://github.com/downloads/quag/mcobj/4x4.jpg)
