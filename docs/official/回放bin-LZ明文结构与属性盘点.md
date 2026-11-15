# 回放 bin LZ 明文结构与属性帧盘点（2026-11-15）

> lz_exact（docs/official/fxnet2-LZ压缩方言全量定案.md）落地后的镜像数据字典：
> A 节=19 个 LZ 镜像帧逐帧明文结构；B 节=0x10 属性帧全量清单。
> bin=replay_p1_slim.bin(13帧)/replay_p2_slim.bin(322帧)，帧格式 [u32 len][frame]。

---

## A. LZ 镜像帧明文结构（19 帧，全部独立池可解、wrap_compress 重编码逐字节一致）

| 帧位 | op | 明文长 | 线帧长 | 内层结构 |
|---|---|---|---|---|
| P1#0 | 0x27 | 9099 | 1276 | 压缩 custommsg m368(on_custom_interact_msg) TVarList 231 值 |
| P1#2 | 0x27 | 3527 | 2337 | 压缩 custommsg m414(sub_11044954) TVarList 704 值 |
| P1#6 | 0x28 | 5131 | 3187 | 化身玩家档案（41 字段+技能行区，走查=外服玩家档案0x28全量帧.md；头 16B: a40f0207c33c3b00fe146844e8fb4242） |
| P2#6 | 0x29 | 285 | 98 | ServerRecordAddRow t=1 串表形态 [0x74][1][0x1a7][10]，20 串 |
| P2#80 | 0x2a | 335 | 325 | ServerObjectProperty obj=1 own=1 row=34 [181][08]，33 属性对 |
| P2#81 | 0x2a | 297 | 290 | ServerObjectProperty obj=1 own=3 row=32 [181][08]，31 属性对 |
| P2#82 | 0x2a | 299 | 283 | ServerObjectProperty obj=1 own=4 row=32 [181][08]，31 属性对 |
| P2#83 | 0x2a | 298 | 290 | ServerObjectProperty obj=1 own=6 row=32 [181][08]，31 属性对 |
| P2#84 | 0x2a | 267 | 261 | ServerObjectProperty obj=1 own=7 row=30 [181][08]，29 属性对 |
| P2#85 | 0x2a | 341 | 315 | ServerObjectProperty obj=1 own=8 row=32 [181][08]，31 属性对 |
| P2#86 | 0x2a | 271 | 265 | ServerObjectProperty obj=1 own=11 row=30 [181][08]，29 属性对 |
| P2#87 | 0x2a | 271 | 263 | ServerObjectProperty obj=1 own=12 row=30 [181][08]，29 属性对 |
| P2#88 | 0x2a | 275 | 263 | ServerObjectProperty obj=1 own=13 row=30 [181][08]，29 属性对 |
| P2#89 | 0x2a | 300 | 287 | ServerObjectProperty obj=1 own=14 row=32 [181][08]，31 属性对 |
| P2#90 | 0x2a | 300 | 278 | ServerObjectProperty obj=1 own=15 row=32 [181][08]，31 属性对 |
| P2#91 | 0x2a | 325 | 303 | ServerObjectProperty obj=1 own=22 row=34 [181][08]，33 属性对 |
| P2#113 | 0x2a | 277 | 263 | ServerObjectProperty obj=121 own=22 row=31 (无 181/08 头)，31 属性对 |
| P2#114 | 0x2a | 321 | 303 | ServerObjectProperty obj=121 own=25 row=33 [181][08]，32 属性对 |
| P2#115 | 0x2a | 317 | 302 | ServerObjectProperty obj=121 own=26 row=33 [181][08]，32 属性对 |

### 内层值明细

**P1#0 op=0x27（压缩 custommsg m368(on_custom_interact_msg) TVarList 231 值，明文 9099B）**

```
  tag2: 368
  tag2: 1
  tag2: 684
  tag6: "0;0;5;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len219)
  tag6: "0;0;20;0;0,00:00-24:00,1,00:00-24:00,2,0…"(len194)
  tag6: ""
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;120211;5;0;1,18:55-19:30,2,18:55-19:30…"(len109)
  tag6: "0;120211;5;0;1,18:55-19:30,2,18:55-19:30…"(len109)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: ""
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;1;0;6,20:00-21:05;6;"
  tag6: ""
  tag6: "0;0;2;1;5,21:30-22:00,6,21:30-22:00;5,6;"
  tag6: ""
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,20:00-22:00,1,20:00-22:00,2,20…"(len120)
  tag6: "0;0;1;1;0,21:30-24:00;0;"
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,10:00-10:30#15:00-15:30#21:30-…"(len168)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0;;"
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;0,09:00-24:00,1,09:00-24:00,2,09…"(len120)
  tag6: "0;0;4;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;1,18:00-24:00,2,18:00-24:00,3,18…"(len72)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;3;0;0,10:00-10:30#15:00-15:30#21:30-…"(len168)
  tag6: "0;115321;0;0;0,00:00-24:00,1,00:00-24:00…"(len125)
  tag6: ""
  tag6: "0;0;0;0;0,15:00-18:00,1,21:20-24:00,3,21…"(len56)
  tag6: "0;0;0;1;2,21:20-24:00,4,21:20-24:00,6,15…"(len56)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: ""
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;;;"
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;5,18:00-20:00,0,18:00-20:00;5,0;"
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;0,10:00-14:00#18:00-24:00,1,10:0…"(len204)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;5,20:00-24:00;5;"
  tag6: "0;0;0;0;0,14:00-23:00;0;"
  tag6: "0;0;0;0;;;"
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;2;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,12:00-23:50,1,12:00-23:50,2,12…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;107055;5;0;0,00:00-24:00,1,00:00-24:00…"(len125)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: ""
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;100;0;0,00:00-24:00,1,00:00-24:00,2,…"(len213)
  tag6: "0;0;100;0;0,00:00-24:00,1,00:00-24:00,2,…"(len213)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;0,00:00-24:00,1,00:00-24:00,2,00…"(len120)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;999;0;0,00:00-24:00,1,00:00-24:00,2,…"(len122)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;0,12:00-23:00,1,12:00-23:00,2,12…"(len72)
  tag6: ""
  tag6: ""
  tag6: "0;116359;0;0;0,12:00-20:30,4,12:00-23:00…"(len77)
  tag6: ""
  tag6: "0;0;0;0;0,12:00-20:30,1,12:00-23:00,2,12…"(len88)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;0;0;0;0,12:00-20:30,4,12:00-23:00,5,12…"(len72)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: "0;118610;0;0;0,18:00-24:00,1,18:00-24:00…"(len125)
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
  tag6: ""
```

**P1#2 op=0x27（压缩 custommsg m414(sub_11044954) TVarList 704 值，明文 3527B）**

```
  tag2: 414
  tag6: "sync"
  tag2: 2
  tag2: 3
  tag2: 5
  tag2: 6
  tag2: 10
  tag2: 12
  tag2: 14
  tag2: 15
  tag2: 16
  tag2: 19
  tag2: 21
  tag2: 22
  tag2: 24
  tag2: 25
  tag2: 31
  tag2: 32
  tag2: 33
  tag2: 34
  tag2: 39
  tag2: 41
  tag2: 42
  tag2: 44
  tag2: 45
  tag2: 46
  tag2: 47
  tag2: 48
  tag2: 50
  tag2: 51
  tag2: 52
  tag2: 53
  tag2: 54
  tag2: 55
  tag2: 56
  tag2: 60
  tag2: 61
  tag2: 66
  tag2: 68
  tag2: 69
  tag2: 70
  tag2: 74
  tag2: 76
  tag2: 79
  tag2: 80
  tag2: 82
  tag2: 83
  tag2: 85
  tag2: 88
  tag2: 89
  tag2: 90
  tag2: 91
  tag2: 92
  tag2: 93
  tag2: 94
  tag2: 97
  tag2: 99
  tag2: 100
  tag2: 101
  tag2: 104
  tag2: 108
  tag2: 110
  tag2: 112
  tag2: 114
  tag2: 115
  tag2: 116
  tag2: 117
  tag2: 118
  tag2: 119
  tag2: 120
  tag2: 125
  tag2: 134
  tag2: 135
  tag2: 136
  tag2: 146
  tag2: 151
  tag2: 153
  tag2: 155
  tag2: 157
  tag2: 158
  tag2: 159
  tag2: 162
  tag2: 165
  tag2: 171
  tag2: 172
  tag2: 173
  tag2: 179
  tag2: 185
  tag2: 191
  tag2: 207
  tag2: 208
  tag2: 211
  tag2: 212
  tag2: 213
  tag2: 214
  tag2: 215
  tag2: 216
  tag2: 218
  tag2: 219
  tag2: 221
  tag2: 222
  tag2: 223
  tag2: 224
  tag2: 226
  tag2: 227
  tag2: 228
  tag2: 229
  tag2: 230
  tag2: 231
  tag2: 236
  tag2: 247
  tag2: 254
  tag2: 255
  tag2: 257
  tag2: 260
  tag2: 261
  tag2: 262
  tag2: 264
  tag2: 265
  tag2: 266
  tag2: 267
  tag2: 268
  tag2: 269
  tag2: 271
  tag2: 272
  tag2: 273
  tag2: 274
  tag2: 276
  tag2: 277
  tag2: 278
  tag2: 279
  tag2: 280
  tag2: 281
  tag2: 282
  tag2: 283
  tag2: 285
  tag2: 286
  tag2: 287
  tag2: 288
  tag2: 289
  tag2: 292
  tag2: 295
  tag2: 296
  tag2: 297
  tag2: 299
  tag2: 304
  tag2: 307
  tag2: 311
  tag2: 314
  tag2: 315
  tag2: 317
  tag2: 318
  tag2: 329
  tag2: 338
  tag2: 339
  tag2: 340
  tag2: 342
  tag2: 343
  tag2: 345
  tag2: 346
  tag2: 347
  tag2: 348
  tag2: 350
  tag2: 357
  tag2: 358
  tag2: 359
  tag2: 360
  tag2: 361
  tag2: 362
  tag2: 363
  tag2: 364
  tag2: 365
  tag2: 366
  tag2: 367
  tag2: 370
  tag2: 371
  tag2: 372
  tag2: 373
  tag2: 374
  tag2: 375
  tag2: 381
  tag2: 382
  tag2: 384
  tag2: 385
  tag2: 386
  tag2: 387
  tag2: 389
  tag2: 391
  tag2: 394
  tag2: 395
  tag2: 396
  tag2: 397
  tag2: 398
  tag2: 399
  tag2: 502
  tag2: 503
  tag2: 504
  tag2: 505
  tag2: 506
  tag2: 507
  tag2: 508
  tag2: 511
  tag2: 513
  tag2: 514
  tag2: 515
  tag2: 516
  tag2: 517
  tag2: 520
  tag2: 522
  tag2: 525
  tag2: 527
  tag2: 528
  tag2: 529
  tag2: 602
  tag2: 603
  tag2: 605
  tag2: 606
  tag2: 607
  tag2: 609
  tag2: 610
  tag2: 612
  tag2: 613
  tag2: 617
  tag2: 618
  tag2: 621
  tag2: 622
  tag2: 623
  tag2: 625
  tag2: 632
  tag2: 637
  tag2: 638
  tag2: 640
  tag2: 641
  tag2: 643
  tag2: 644
  tag2: 646
  tag2: 648
  tag2: 650
  tag2: 651
  tag2: 655
  tag2: 656
  tag2: 657
  tag2: 658
  tag2: 659
  tag2: 661
  tag2: 662
  tag2: 663
  tag2: 666
  tag2: 667
  tag2: 669
  tag2: 673
  tag2: 675
  tag2: 676
  tag2: 677
  tag2: 678
  tag2: 679
  tag2: 680
  tag2: 681
  tag2: 682
  tag2: 683
  tag2: 684
  tag2: 685
  tag2: 686
  tag2: 687
  tag2: 688
  tag2: 689
  tag2: 690
  tag2: 691
  tag2: 692
  tag2: 693
  tag2: 694
  tag2: 695
  tag2: 696
  tag2: 697
  tag2: 698
  tag2: 700
  tag2: 701
  tag2: 703
  tag2: 711
  tag2: 712
  tag2: 714
  tag2: 715
  tag2: 716
  tag2: 717
  tag2: 724
  tag2: 726
  tag2: 728
  tag2: 729
  tag2: 730
  tag2: 731
  tag2: 732
  tag2: 741
  tag2: 742
  tag2: 743
  tag2: 750
  tag2: 752
  tag2: 753
  tag2: 766
  tag2: 772
  tag2: 774
  tag2: 775
  tag2: 776
  tag2: 780
  tag2: 781
  tag2: 782
  tag2: 783
  tag2: 785
  tag2: 786
  tag2: 787
  tag2: 790
  tag2: 791
  tag2: 800
  tag2: 801
  tag2: 802
  tag2: 803
  tag2: 804
  tag2: 805
  tag2: 806
  tag2: 807
  tag2: 809
  tag2: 810
  tag2: 811
  tag2: 812
  tag2: 813
  tag2: 814
  tag2: 816
  tag2: 817
  tag2: 821
  tag2: 822
  tag2: 851
  tag2: 854
  tag2: 855
  tag2: 857
  tag2: 858
  tag2: 860
  tag2: 861
  tag2: 882
  tag2: 883
  tag2: 884
  tag2: 885
  tag2: 886
  tag2: 894
  tag2: 895
  tag2: 897
  tag2: 898
  tag2: 899
  tag2: 900
  tag2: 901
  tag2: 904
  tag2: 905
  tag2: 906
  tag2: 907
  tag2: 911
  tag2: 912
  tag2: 913
  tag2: 914
  tag2: 915
  tag2: 916
  tag2: 917
  tag2: 918
  tag2: 922
  tag2: 923
  tag2: 924
  tag2: 926
  tag2: 928
  tag2: 930
  tag2: 931
  tag2: 932
  tag2: 933
  tag2: 934
  tag2: 935
  tag2: 936
  tag2: 937
  tag2: 938
  tag2: 954
  tag2: 955
  tag2: 956
  tag2: 958
  tag2: 961
  tag2: 964
  tag2: 965
  tag2: 966
  tag2: 967
  tag2: 971
  tag2: 972
  tag2: 975
  tag2: 977
  tag2: 978
  tag2: 982
  tag2: 983
  tag2: 984
  tag2: 986
  tag2: 989
  tag2: 991
  tag2: 992
  tag2: 996
  tag2: 998
  tag2: 999
  tag2: 1021
  tag2: 1022
  tag2: 1028
  tag2: 1029
  tag2: 1036
  tag2: 1049
  tag2: 1050
  tag2: 1052
  tag2: 1053
  tag2: 1055
  tag2: 1056
  tag2: 1057
  tag2: 1058
  tag2: 1060
  tag2: 1061
  tag2: 1062
  tag2: 1063
  tag2: 1064
  tag2: 1065
  tag2: 1066
  tag2: 1069
  tag2: 1070
  tag2: 1072
  tag2: 1073
  tag2: 1074
  tag2: 1075
  tag2: 1076
  tag2: 1078
  tag2: 1079
  tag2: 1080
  tag2: 1081
  tag2: 1082
  tag2: 1083
  tag2: 1084
  tag2: 1085
  tag2: 1086
  tag2: 1087
  tag2: 1088
  tag2: 1090
  tag2: 1092
  tag2: 1093
  tag2: 1094
  tag2: 1095
  tag2: 1096
  tag2: 1097
  tag2: 1098
  tag2: 1099
  tag2: 1101
  tag2: 1102
  tag2: 1103
  tag2: 1104
  tag2: 1105
  tag2: 1106
  tag2: 1107
  tag2: 1108
  tag2: 1109
  tag2: 1110
  tag2: 1115
  tag2: 1116
  tag2: 1121
  tag2: 1122
  tag2: 1123
  tag2: 1124
  tag2: 1125
  tag2: 1127
  tag2: 1128
  tag2: 1130
  tag2: 1133
  tag2: 1134
  tag2: 1135
  tag2: 1136
  tag2: 1138
  tag2: 1141
  tag2: 1142
  tag2: 1143
  tag2: 1149
  tag2: 1150
  tag2: 1151
  tag2: 1152
  tag2: 1154
  tag2: 1156
  tag2: 1157
  tag2: 1165
  tag2: 1166
  tag2: 1167
  tag2: 1168
  tag2: 1169
  tag2: 1170
  tag2: 1171
  tag2: 1172
  tag2: 1173
  tag2: 1174
  tag2: 1178
  tag2: 1181
  tag2: 1183
  tag2: 1184
  tag2: 1185
  tag2: 1192
  tag2: 1193
  tag2: 1195
  tag2: 1196
  tag2: 1206
  tag2: 1210
  tag2: 1211
  tag2: 1214
  tag2: 1218
  tag2: 1221
  tag2: 1222
  tag2: 1224
  tag2: 1250
  tag2: 1251
  tag2: 1259
  tag2: 1260
  tag2: 1263
  tag2: 1265
  tag2: 1266
  tag2: 1267
  tag2: 1268
  tag2: 1270
  tag2: 1271
  tag2: 1272
  tag2: 1273
  tag2: 1275
  tag2: 1277
  tag2: 1278
  tag2: 1279
  tag2: 1284
  tag2: 1285
  tag2: 1286
  tag2: 1288
  tag2: 1289
  tag2: 1290
  tag2: 1291
  tag2: 1292
  tag2: 1293
  tag2: 1294
  tag2: 1296
  tag2: 1297
  tag2: 1298
  tag2: 1299
  tag2: 1300
  tag2: 1301
  tag2: 1311
  tag2: 1312
  tag2: 1313
  tag2: 1317
  tag2: 1318
  tag2: 1319
  tag2: 1320
  tag2: 1321
  tag2: 1322
  tag2: 1323
  tag2: 1324
  tag2: 1328
  tag2: 1329
  tag2: 1330
  tag2: 1334
  tag2: 1336
  tag2: 1345
  tag2: 1347
  tag2: 1349
  tag2: 1350
  tag2: 1353
  tag2: 1354
  tag2: 1356
  tag2: 1357
  tag2: 1359
  tag2: 1362
  tag2: 1363
  tag2: 1364
  tag2: 1365
  tag2: 1367
  tag2: 1368
  tag2: 1369
  tag2: 1370
  tag2: 1371
  tag2: 1372
  tag2: 1373
  tag2: 1374
  tag2: 1375
  tag2: 1381
  tag2: 1382
  tag2: 1384
  tag2: 1385
  tag2: 1386
  tag2: 1387
  tag2: 1388
  tag2: 1389
  tag2: 1396
  tag2: 1397
  tag2: 1398
  tag2: 1401
  tag2: 1402
  tag2: 1404
  tag2: 1405
  tag2: 1407
  tag2: 1408
  tag2: 1410
  tag2: 1411
  tag2: 1412
  tag2: 1504
  tag2: 1506
  tag2: 1509
  tag2: 1510
  tag2: 2005
  tag2: 2008
  tag2: 2016
  tag2: 2017
  tag2: 2020
  tag2: 2029
  tag2: 2031
  tag2: 2037
  tag2: 2042
  tag2: 2043
  tag2: 2044
  tag2: 2045
  tag2: 2046
  tag2: 2049
  tag2: 2050
  tag2: 2051
  tag2: 2052
  tag2: 2053
  tag2: 2056
  tag2: 2058
  tag2: 2061
  tag2: 2111
  tag2: 2115
  tag2: 2116
  tag2: 2118
  tag2: 2127
  tag2: 2128
  tag2: 2131
  tag2: 2132
  tag2: 2134
  tag2: 2135
  tag2: 2136
  tag2: 2137
  tag2: 2138
  tag2: 2139
  tag2: 2140
  tag2: 2141
  tag2: 2142
  tag2: 2143
  tag2: 2146
  tag2: 2148
  tag2: 2505
  tag2: 2506
  tag2: 2811
  tag2: 2812
  tag2: 2816
  tag2: 2820
  tag2: 2822
  tag2: 2823
  tag2: 2824
  tag2: 2831
  tag2: 2835
  tag2: 2836
  tag2: 2837
  tag2: 2838
  tag2: 2841
  tag2: 2842
  tag2: 2843
  tag2: 2845
  tag2: 2846
  tag2: 2849
  tag2: 2850
  tag2: 2852
  tag2: 2860
  tag2: 2861
  tag2: 2863
  tag2: 2865
  tag2: 2866
  tag2: 2869
  tag2: 2872
  tag2: 2873
  tag2: 2874
  tag2: 2877
  tag2: 2878
  tag2: 2879
  tag2: 2880
  tag2: 2882
  tag2: 2883
  tag2: 2885
  tag2: 2886
  tag2: 2887
  tag2: 2890
  tag2: 2892
  tag2: 2893
  tag2: 2894
  tag2: 2896
  tag2: 2897
  tag2: 2899
  tag2: 2901
  tag2: 2903
  tag2: 2907
  tag2: 2917
  tag2: 2918
  tag2: 2919
  tag2: 2922
  tag2: 2924
  tag2: 2925
  tag2: 2927
  tag2: 2931
  tag2: 2932
  tag2: 2933
  tag2: 2938
```

**P2#6 op=0x29（ServerRecordAddRow t=1 串表形态 [0x74][1][0x1a7][10]，20 串，明文 285B）**

```
  串: gemskill_052
  串: sh_ys
  串: gemskill_058
  串: sh_ys
  串: gemskill_059
  串: sh_ys
  串: gemskill_043
  串: sh_ys
  串: gemskill_044
  串: sh_ys
  串: gemskill_012
  串: sh_ys
  串: gemskill_026
  串: sh_ys
  串: gemskill_064
  串: sh_ys
  串: gemskill_062
  串: sh_ys
  串: gemskill_014
  串: sh_ys
```

**P2#80 op=0x2a（ServerObjectProperty obj=1 own=1 row=34 [181][08]，33 属性对，明文 335B）**

```
  1447(ConfigID)="helmet_60705"
  1896(ItemType)=140
  1898(ViewID)=2
  1899(ColorLevel)=6
  1900(UniqueID)="7100037-014-1497590557-0645"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=247698
  1920(LogicPack)=3
  1921(ArtPack)=50395
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=94
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Hat"
  586(PhyDef)=166
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=24408
  1960(PropGetPack)=8386
  1961(PropBuilt)="1,62948;1,64627;1,62952;1,64727;1,62950;1,64722;"
  1963(LifeSplitItemPackage)=305
  1966(EffectPackID)=17217
  1967(ReplacePack)=14233
  1968(ReplaceID)="Tab_b_head_180_b"
  1974(TaoZhuangID)=10031
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_hel_01,1,eqp_test23,1"
```

**P2#81 op=0x2a（ServerObjectProperty obj=1 own=3 row=32 [181][08]，31 属性对，明文 297B）**

```
  1447(ConfigID)="cloth_50704"
  1896(ItemType)=141
  1898(ViewID)=2
  1899(ColorLevel)=5
  1900(UniqueID)="186100106-013-1459091929-7417"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=116560
  1920(LogicPack)=3
  1921(ArtPack)=50396
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=89
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Cloth"
  586(PhyDef)=148
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=24403
  1960(PropGetPack)=8381
  1961(PropBuilt)="1,62960;1,64726;1,62947;1,64727;1,62950;1,64519;"
  1963(LifeSplitItemPackage)=288
  1966(EffectPackID)=16236
  1974(TaoZhuangID)=10029
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_nclo_03,2"
```

**P2#82 op=0x2a（ServerObjectProperty obj=1 own=4 row=32 [181][08]，31 属性对，明文 299B）**

```
  1447(ConfigID)="pants_50704"
  1896(ItemType)=142
  1898(ViewID)=2
  1899(ColorLevel)=5
  1900(UniqueID)="186100106-013-1457630485-3388"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=116560
  1920(LogicPack)=3
  1921(ArtPack)=50398
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=89
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Pants"
  586(PhyDef)=148
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=23863
  1960(PropGetPack)=8381
  1961(PropBuilt)="1,62955;1,64626;1,62958;1,64724;1,62955;1,64526;"
  1963(LifeSplitItemPackage)=288
  1966(EffectPackID)=16229
  1974(TaoZhuangID)=10029
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_pants_01a,2"
```

**P2#83 op=0x2a（ServerObjectProperty obj=1 own=6 row=32 [181][08]，31 属性对，明文 298B）**

```
  1447(ConfigID)="wrist_50704"
  1896(ItemType)=145
  1898(ViewID)=2
  1899(ColorLevel)=5
  1900(UniqueID)="186100106-013-1456457060-0037"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=87420
  1920(LogicPack)=3
  1921(ArtPack)=50220
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=89
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Wrist"
  586(PhyDef)=148
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=26293
  1960(PropGetPack)=9101
  1961(PropBuilt)="1,62955;1,64724;1,62960;1,64426;1,62947;1,64324;"
  1963(LifeSplitItemPackage)=288
  1966(EffectPackID)=16237
  1974(TaoZhuangID)=10029
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_wrist_01,2"
```

**P2#84 op=0x2a（ServerObjectProperty obj=1 own=7 row=30 [181][08]，29 属性对，明文 267B）**

```
  1447(ConfigID)="leg_50704"
  1896(ItemType)=144
  1898(ViewID)=2
  1899(ColorLevel)=5
  1900(UniqueID)="186100106-013-1455301355-8956"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=87420
  1920(LogicPack)=3
  1921(ArtPack)=50210
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=89
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Leg"
  586(PhyDef)=148
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=27283
  1960(PropGetPack)=9416
  1961(PropBuilt)="1,62960;1,64324;1,62958;1,64726;1,62947;1,64726;"
  1963(LifeSplitItemPackage)=288
  1974(TaoZhuangID)=10029
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
```

**P2#85 op=0x2a（ServerObjectProperty obj=1 own=8 row=32 [181][08]，31 属性对，明文 341B）**

```
  1447(ConfigID)="shoes_60602"
  1896(ItemType)=143
  1898(ViewID)=2
  1899(ColorLevel)=6
  1900(UniqueID)="186100106-014-1426535108-9699"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=203203
  1920(LogicPack)=3
  1921(ArtPack)=1835
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=89
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Shoes"
  586(PhyDef)=135
  1472(Level)=61
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1961(PropBuilt)="1,62956;1,64523;1,62946;1,64421;1,62954;1,64725;"
  1963(LifeSplitItemPackage)=303
  1966(EffectPackID)=17226
  1967(ReplacePack)=41561
  1968(ReplaceID)="Tab_b_shoes_battlefield_60601"
  1974(TaoZhuangID)=10005
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_shoes_02,1,eqp_test21,1"
```

**P2#86 op=0x2a（ServerObjectProperty obj=1 own=11 row=30 [181][08]，29 属性对，明文 271B）**

```
  1447(ConfigID)="ear_50704"
  1896(ItemType)=161
  1898(ViewID)=2
  1899(ColorLevel)=5
  1900(UniqueID)="186100106-013-1449726986-6211"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=134044
  1920(LogicPack)=3
  1921(ArtPack)=50250
  1923(KarmaModifyPack)=35
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=94
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Earring"
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=24043
  1960(PropGetPack)=8426
  1961(PropBuilt)="1,15505;1,64716;1,15504;1,64311;1,15504;1,64714;"
  1963(LifeSplitItemPackage)=186
  1974(TaoZhuangID)=10029
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
```

**P2#87 op=0x2a（ServerObjectProperty obj=1 own=12 row=30 [181][08]，29 属性对，明文 271B）**

```
  1447(ConfigID)="ear_50704"
  1896(ItemType)=161
  1898(ViewID)=2
  1899(ColorLevel)=5
  1900(UniqueID)="186100106-004-1449039053-5918"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=134044
  1920(LogicPack)=3
  1921(ArtPack)=50250
  1923(KarmaModifyPack)=35
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=91
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Earring"
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=24583
  1960(PropGetPack)=8426
  1961(PropBuilt)="1,15504;1,64416;1,15505;1,64716;1,15504;1,64712;"
  1963(LifeSplitItemPackage)=186
  1974(TaoZhuangID)=10029
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
```

**P2#88 op=0x2a（ServerObjectProperty obj=1 own=13 row=30 [181][08]，29 属性对，明文 275B）**

```
  1447(ConfigID)="amulet_50704"
  1896(ItemType)=160
  1898(ViewID)=2
  1899(ColorLevel)=5
  1900(UniqueID)="186100106-004-1453254374-1157"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=139872
  1920(LogicPack)=3
  1921(ArtPack)=50230
  1923(KarmaModifyPack)=39
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=98
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Necklace"
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=24223
  1960(PropGetPack)=8426
  1961(PropBuilt)="1,15505;1,64721;1,15505;1,64520;1,15504;1,64720;"
  1963(LifeSplitItemPackage)=186
  1974(TaoZhuangID)=10029
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
```

**P2#89 op=0x2a（ServerObjectProperty obj=1 own=14 row=32 [181][08]，31 属性对，明文 300B）**

```
  1447(ConfigID)="ring_60701"
  1896(ItemType)=162
  1898(ViewID)=2
  1899(ColorLevel)=6
  1900(UniqueID)="186100106-013-1454159668-9640"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=335120
  1920(LogicPack)=3
  1921(ArtPack)=50242
  1923(KarmaModifyPack)=20
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=89
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Fingerring"
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=24318
  1960(PropGetPack)=8431
  1961(PropBuilt)="1,15505;1,64320;1,15505;1,64720;1,15506;1,64518;"
  1963(LifeSplitItemPackage)=203
  1966(EffectPackID)=11151
  1974(TaoZhuangID)=10001
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_test23,1"
```

**P2#90 op=0x2a（ServerObjectProperty obj=1 own=15 row=32 [181][08]，31 属性对，明文 300B）**

```
  1447(ConfigID)="ring_60701"
  1896(ItemType)=162
  1898(ViewID)=2
  1899(ColorLevel)=6
  1900(UniqueID)="186100106-004-1446149209-1143"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=335120
  1920(LogicPack)=3
  1921(ArtPack)=50242
  1923(KarmaModifyPack)=20
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=82
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Fingerring"
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=24048
  1960(PropGetPack)=8431
  1961(PropBuilt)="1,15506;1,64420;1,15506;1,64320;1,15506;1,64715;"
  1963(LifeSplitItemPackage)=203
  1966(EffectPackID)=11151
  1974(TaoZhuangID)=10001
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_test23,1"
```

**P2#91 op=0x2a（ServerObjectProperty obj=1 own=22 row=34 [181][08]，33 属性对，明文 325B）**

```
  1447(ConfigID)="sword_60701"
  1896(ItemType)=102
  1898(ViewID)=2
  1899(ColorLevel)=6
  1900(UniqueID)="186100106-004-1446136584-5835"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=378832
  1920(LogicPack)=3
  1921(ArtPack)=50320
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1942(Hardiness)=97
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Weapon"
  511(MaxMeleeDamage)=65
  512(MinMeleeDamage)=38
  1472(Level)=71
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=6
  1959(RandomRulePack)=29718
  1960(PropGetPack)=8476
  1961(PropBuilt)="1,13063;1,13058;1,13054;1,13058;1,13068;1,13066;"
  1962(CanBePoisoned)=1
  1963(LifeSplitItemPackage)=101
  1966(EffectPackID)=1647
  1974(TaoZhuangID)=10001
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_WpResParry,6,eqp_wp_06,1"
```

**P2#113 op=0x2a（ServerObjectProperty obj=121 own=22 row=31 (无 181/08 头)，31 属性对，明文 277B）**

```
  1942(Hardiness)=95
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="ShotWeapon"
  513(MaxShotDamage)=46
  514(MinShotDamage)=26
  1472(Level)=61
  1956(IsCanBuild)=1
  1957(MaxBuildCount)=6
  1958(CurBuildCount)=5
  1959(RandomRulePack)=8234
  1960(PropGetPack)=3284
  1961(PropBuilt)="1,15020;1,15019;1,15017;1,15017;1,15028;"
  1963(LifeSplitItemPackage)=82
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  181(Type)=8
  1447(ConfigID)="hw_claws_50601"
  1896(ItemType)=116
  1898(ViewID)=2
  1899(ColorLevel)=5
  1900(UniqueID)="186100106-014-1388788718-2228"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=124313
  1920(LogicPack)=3
  1921(ArtPack)=44110
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
```

**P2#114 op=0x2a（ServerObjectProperty obj=121 own=25 row=33 [181][08]，32 属性对，明文 321B）**

```
  1447(ConfigID)="sthorn_40403"
  1896(ItemType)=106
  1898(ViewID)=2
  1899(ColorLevel)=4
  1900(UniqueID)="186100082-014-1354026496-6827"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=40706
  1920(LogicPack)=3
  1921(ArtPack)=614
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1893(CompositePlayerName)="接芷若(涴)"
  1942(Hardiness)=68
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Weapon"
  511(MaxMeleeDamage)=26
  512(MinMeleeDamage)=18
  1472(Level)=41
  1957(MaxBuildCount)=4
  1958(CurBuildCount)=4
  1960(PropGetPack)=1210
  1961(PropBuilt)="1,13252;1,13242;1,13248;1,13242;"
  1962(CanBePoisoned)=1
  1963(LifeSplitItemPackage)=61
  1966(EffectPackID)=1011
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_WpResParry,4,eqp_AddBlood,1"
```

**P2#115 op=0x2a（ServerObjectProperty obj=121 own=26 row=33 [181][08]，32 属性对，明文 317B）**

```
  1447(ConfigID)="ssword_40405"
  1896(ItemType)=105
  1898(ViewID)=2
  1899(ColorLevel)=4
  1900(UniqueID)="186100082-014-1353647421-0467"
  1901(Amount)=1
  1902(MaxAmount)=1
  1905(BindStatus)=1
  1908(SellPrice1)=40706
  1920(LogicPack)=3
  1921(ArtPack)=11008
  1933(IsCanConsign)=1
  1934(IsMarketItem)=1
  1935(IsCarryItem)=1
  1893(CompositePlayerName)="大水"
  1942(Hardiness)=76
  1943(MaxHardiness)=100
  1944(NeedSex)=2
  1945(EquipType)="Weapon"
  511(MaxMeleeDamage)=27
  512(MinMeleeDamage)=17
  1472(Level)=41
  1957(MaxBuildCount)=4
  1958(CurBuildCount)=4
  1960(PropGetPack)=1210
  1961(PropBuilt)="1,13048;1,13051;1,13044;1,13049;"
  1962(CanBePoisoned)=1
  1963(LifeSplitItemPackage)=61
  1966(EffectPackID)=1008
  1976(MaxProficiency)=4000
  1977(EquipDrop)=1
  1991(EquipEffectString)="eqp_WpResParry,4,eqp_AddParrySlow,1"
```

---

## B. 0x10 属性帧盘点（119 帧，残差纪律校验）

全部帧逐行解析恰好耗尽（残差=0），宽度纪律与 2433 表逐属性吻合。

共 107 个不同 idx：

| idx | 2433 列名 | 型 | 值（跨帧） | 帧位 |
|---|---|---|---|---|
| 65 | IsGuildCaptain | Int32 | 1 | P2#56 |
| 158 | VisualPlayers | Word | 210 | P2#4 |
| 221 | ModifySkillLockTime | Int32 | 200 | P2#77 |
| 234 | AutoTrace | Int32 | 1 | P2#98 |
| 248 | SafeLoginFlag | Int32 | 6 | P1#8 |
| 251 | YinLevel | Int32 | 1089 | P2#71 |
| 252 | YangLevel | Int32 | 623 | P2#72 |
| 253 | TaijiLevel | Int32 | 299 | P2#73 |
| 257 | ProRefreshFlag | Int32 | 1 | P2#236 |
| 304 | IsMobileIDBind | Byte | 1 | P2#311 |
| 386 | MoveSpeed | Float32 | 4.5 | P2#224 |
| 390 | RunSpeedAdd | Float32 | 0.6 | P2#225 |
| 391 | DropHeightPub | Float32 | 24 | P2#230 |
| 471 | HPRatio | Int32 | 100 | P2#233 |
| 472 | MPRatio | Int32 | 100 | P2#235 |
| 478 | MaxParry | Int32 | 10372 | P2#217 |
| 479 | MaxParryAdd | Int32 | 9486 | P2#168 |
| 480 | ParryRatio | Int32 | 100 | P2#216 |
| 481 | ParryUpSpeed | Int32 | 104 | P2#218 |
| 482 | ParryUpSpeedAdd | Int32 | 10 | P2#169 |
| 484 | HitHPRatio | Int32 | 100 | P2#234 |
| 492 | StrAdd | Int32 | 341 | P2#149 |
| 493 | IngAdd | Int32 | 931 | P2#151 |
| 494 | DexAdd | Int32 | 160 | P2#150 |
| 495 | SpiAdd | Int32 | 254 | P2#152 |
| 496 | StaAdd | Int32 | 483 | P2#153 |
| 502 | MeleePower | Int32 | 344 | P2#206 |
| 503 | ShotPower | Int32 | 163 | P2#207 |
| 504 | MagicPower | Int32 | 934 | P2#208 |
| 505 | MeleePowerAdd | Int32 | 247 | P2#158 |
| 506 | ShotPowerAdd | Int32 | 181 | P2#159 |
| 507 | MagicPowerAdd | Int32 | 206 | P2#160 |
| 511 | MaxMeleeDamage | Int32 | 307 | P2#220 |
| 512 | MinMeleeDamage | Int32 | 280 | P2#221 |
| 513 | MaxShotDamage | Int32 | 4 | P2#222 |
| 514 | MinShotDamage | Int32 | 4 | P2#223 |
| 516 | StiDamageAdd | Int32 | 130 / 77 / 59 / 114 | P2#144 P2#145 P2#146 P2#147 |
| 517 | JujDamageAdd | Int32 | 35 / 16 / 4 | P2#141 P2#142 P2#143 |
| 520 | PhyDamageMul | Float32 | 1.1 / 1 | P2#140 P2#148 P2#170 |
| 521 | StiDamageMul | Float32 | 1 | P2#171 |
| 522 | JujDamageMul | Float32 | 1 | P2#172 |
| 523 | NegDamageMul | Float32 | 1 | P2#173 |
| 524 | MasDamageMul | Float32 | 1 | P2#174 |
| 525 | PhyFinalDamageAdd | Int32 | 1344 | P2#199 |
| 526 | StiFinalDamageAdd | Int32 | 128 | P2#200 |
| 527 | JujFinalDamageAdd | Int32 | 144 | P2#201 |
| 540 | AIPhyDefAdd | Int32 | 480 | P2#183 |
| 541 | AIMagicDefAdd | Int32 | 456 | P2#185 |
| 542 | PIPhyDefAdd | Float32 | 2 | P2#184 |
| 548 | AIPhyParryAdd | Float32 | 1 / 11 | P2#190 P2#319 |
| 549 | AIMagicParryAdd | Float32 | 30 | P2#191 |
| 550 | PhyParryDamResAdd | Float32 | 9 | P2#192 |
| 551 | MagicParryDamResAdd | Float32 | 25 | P2#193 |
| 552 | AIStiDefAdd | Int32 | 9 | P2#194 |
| 553 | AIJujDefAdd | Int32 | 32 | P2#195 |
| 558 | PhyHit | Int32 | 163 | P2#209 |
| 559 | PhyHitAdd | Int32 | 182 | P2#161 |
| 561 | MagicHit | Int32 | 257 | P2#210 |
| 562 | MagicHitAdd | Int32 | 314 | P2#162 |
| 564 | Dodge | Int32 | 163 | P2#211 |
| 565 | DodgeAdd | Int32 | 109 | P2#163 |
| 566 | DodgeProbAdd | Float32 | 8 | P2#179 |
| 567 | PhyVa | Int32 | 344 | P2#212 |
| 568 | PhyVaAdd | Int32 | 110 | P2#164 |
| 571 | MagicVa | Int32 | 257 | P2#213 |
| 572 | MagicVaAdd | Int32 | 288 | P2#165 |
| 573 | MagicVaProbAdd | Float32 | 4 | P2#180 |
| 574 | MagicVaDamageAdd | Float32 | 30 | P2#181 |
| 575 | VaDef | Int32 | 243 | P2#214 |
| 576 | VaDefAdd | Int32 | 164 | P2#166 |
| 578 | VaDamageDefAdd | Float32 | 76 | P2#182 |
| 580 | MaxHP | Int32 | 4160 | P2#202 |
| 581 | MaxMP | Int32 | 4033 | P2#203 |
| 582 | MaxSP | Int32 | 50 / 130 | P2#60 P2#61 |
| 583 | MaxHPAdd | Int32 | 24125 | P2#154 |
| 584 | MaxMPAdd | Int32 | 6355 | P2#155 |
| 586 | PhyDef | Int32 | 7448 | P2#219 |
| 587 | MinMagicDef | Int32 | 51 | P2#215 |
| 599 | MinMagicDefAdd | Int32 | 802 | P2#167 |
| 600 | StiDefAdd | Int32 | 9 | P2#175 |
| 601 | JujDefAdd | Int32 | 235 | P2#176 |
| 602 | NegDefAdd | Int32 | 510 | P2#177 |
| 603 | MasDefAdd | Int32 | 9 | P2#178 |
| 610 | HPResumeMul | Float32 | 1 | P2#188 |
| 611 | MPResumeMul | Float32 | 1 | P2#189 |
| 615 | HPResumeOnceMul | Float32 | 1 | P2#197 |
| 616 | MPResumeOnceMul | Float32 | 1 | P2#198 |
| 617 | HPUpSpeed | Int32 | 44 | P2#204 |
| 618 | MPUpSpeed | Int32 | 43 | P2#205 |
| 621 | HPUpSpeedAdd | Int32 | 38 | P2#156 |
| 622 | MPUpSpeedAdd | Int32 | 57 | P2#157 |
| 624 | MPHeartSpeedAdd | Int32 | -350 | P2#196 |
| 632 | BufferInfo2 | String | "1E37,7020FA4,3B3CC3,0,0,0" | P2#54 |
| 633 | BufferInfo3 | String | "4BF8,7020FA4,3B3CC3,5,0,0" | P2#69 |
| 656 | CurNeiGongWuXing | Int32 | 6 | P2#68 |
| 662 | DriftJumpSpeedAdd | Float32 | 3 | P2#226 |
| 664 | SndJumpSpeedAdd | Float32 | 4.8 | P2#227 |
| 666 | ThdJumpSpeedAdd | Float32 | 3 | P2#228 |
| 670 | AirRushDistAdd | Float32 | 5 | P2#229 |
| 694 | QingGongPoint | Int32 | 150 / 160 | P1#11 P2#320 |
| 696 | MaxQingGongPointAdd | Int32 | 15 | P2#186 |
| 811 | LiveSpeedPower | Int32 | 7 | P2#187 |
| 828 | UnderWaterModifySpeed | Float32 | 1 | P2#231 P2#232 |
| 857 | ZQPLValue | Int32 | 2673440 | P2#64 |
| 864 | WuJiPoint | Int32 | 81 | P2#59 |
| 865 | MaxWuJiPoint | Int32 | 95 | P2#58 |
| 1405 | TodayCapitalLeave1 | Int64 | 360000 / 352000 | P1#3 P1#4 |

---

## C. 替换指引

- A/B 两节即镜像帧再生的数据字典：解码（lz_exact）→ 按 B 节列型/ A 节语法改载荷 →
  `wrap_compress` 重编码 → 重打包 bin（[u32 len][frame]）→ 重编译内嵌。
- **0x10 属性帧已切「真值优先/影子兜底」（2026-11-15 落地）**：`cmd/protocol-probe/replay_attr_mirror.go`
  在帧替换档内逐行换值——self 帧实测 53/106 帧换为我方值（值源=stateProperties 打底
  +combatPanelProperties 覆盖），其余 50 idx 影子默认值兜底（492-496 五维Add/520-527 分系
  增伤/540+ AI 族/610-616 回复倍率等无我方源，481/624 占位常量显式保留影子）；
  影子行重编码字节零漂移由 round-trip 闸保证（TestReplayAttrShadowRoundTrip）。
- 0x2A 装备外观行（P2#80-91/#113-115）已可按 [idx][2433 列型值] 逐对再生，
  DROP_ATTR 差集（52 重叠/53 独有）可据此清单对账销项。
- P1#0/P1#2 为压缩 custommsg（m368/m414），值域语义另行定案；0x28 化身档案
  走查见专文（外服玩家档案0x28全量帧.md）。

