-- สร้าง DB ถ้ายังไม่มี
IF DB_ID('GoDemoDB') IS NULL
BEGIN
  CREATE DATABASE GoDemoDB;
END
GO
USE GoDemoDB;
GO

IF OBJECT_ID('dbo.users','U') IS NULL
BEGIN
  CREATE TABLE dbo.users (
    id            INT IDENTITY(1,1) PRIMARY KEY,
    person_code   NVARCHAR(50) UNIQUE,
    name          NVARCHAR(120) NOT NULL,
    email         NVARCHAR(200) NOT NULL UNIQUE,
    role          NVARCHAR(50) NULL,
    password_hash VARBINARY(256) NULL,
    position      NVARCHAR(120) NULL,
    department    NVARCHAR(120) NULL,
    url_image     NVARCHAR(255) NULL,
    start_date    DATE NULL,
    confirm_date  DATE NULL,
    years_of_work INT NOT NULL DEFAULT 0,
    created_at    DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at    DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME()
  );
END
GO

-- (ออปชัน) บริษัท & mapping
IF OBJECT_ID('dbo.companies','U') IS NULL
BEGIN
  CREATE TABLE dbo.companies (
    id    UNIQUEIDENTIFIER DEFAULT NEWID() PRIMARY KEY,
    code  NVARCHAR(50) NOT NULL,
    name  NVARCHAR(200) NOT NULL,
    image NVARCHAR(255) NULL
  );
END
GO

IF OBJECT_ID('dbo.user_companies','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_companies (
    user_id    INT NOT NULL FOREIGN KEY REFERENCES dbo.users(id) ON DELETE CASCADE,
    company_id UNIQUEIDENTIFIER NOT NULL FOREIGN KEY REFERENCES dbo.companies(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, company_id)
  );
END
GO


-- --- Time Attendance per user (summary + array items) ---
IF OBJECT_ID('dbo.user_time_attendance','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_time_attendance (
    user_id        INT NOT NULL PRIMARY KEY
  , score_max      INT NOT NULL DEFAULT 0
  , score_obtained INT NOT NULL DEFAULT 0
  , penalty_total  INT NOT NULL DEFAULT 0
  , CONSTRAINT fk_uta_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
  );
END
GO

IF OBJECT_ID('dbo.user_time_attendance_items','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_time_attendance_items (
    user_id INT NOT NULL
  , name    NVARCHAR(64) NOT NULL
  , value   NVARCHAR(32) NOT NULL
  , CONSTRAINT pk_uta_items PRIMARY KEY (user_id, name)
  , CONSTRAINT fk_uta_items_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
  );
END
GO


-- ===== Self-Evaluation Form (full fields) =====
IF OBJECT_ID('dbo.eval_form','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_form (
    id               INT IDENTITY(1,1) PRIMARY KEY,
    code             NVARCHAR(50)  NOT NULL UNIQUE,     -- รหัสแบบฟอร์มประเมิน
    title_th         NVARCHAR(200) NOT NULL,            -- แบบฟอร์มการประเมิน (TH)
    title_en         NVARCHAR(200) NULL,                -- แบบฟอร์มการประเมิน (ENG)
    kpi_weight       INT NOT NULL DEFAULT 0,            -- KPI (%)
    comp_weight      INT NOT NULL DEFAULT 0,            -- Competency (%)
    ta_weight        INT NOT NULL DEFAULT 0,            -- Time Attendance (%)
    total_weight     INT NOT NULL DEFAULT 100,          -- Total (100)
    calc_method      TINYINT NOT NULL DEFAULT 1,        -- การคำนวณคะแนน (1=weighted avg, 2=sum capped, ... ปรับภายหลังได้)
    score_scheme     TINYINT NOT NULL DEFAULT 1,        -- รูปแบบการให้คะแนน (1=0-5, 2=0-100, ...)

    -- ช่วงเวลา
    kpi_cfg_start    DATE NULL,   -- วันที่เริ่มการตั้งค่า KPI
    kpi_cfg_end      DATE NULL,   -- วันที่สิ้นสุดการตั้งค่า KPI
    eval_start       DATE NULL,   -- วันที่เริ่มการประเมิน
    eval_end         DATE NULL,   -- วันที่สิ้นสุดการประเมิน
    other_lv_start   DATE NULL,   -- วันที่เริ่มการประเมินลาอื่นๆ
    other_lv_end     DATE NULL,   -- วันที่สิ้นสุดการประเมินลาอื่นๆ
    annual_lv_start  DATE NULL,   -- วันที่เริ่มการประเมินลาพักร้อน
    annual_lv_end    DATE NULL,   -- วันที่สิ้นสุดการประเมินลาพักร้อน

    remark           NVARCHAR(500) NULL,

    created_at       DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at       DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME()
  );
END
GO


-- มอบหมายฟอร์มให้ผู้ใช้ (หนึ่งฟอร์ม ต่อหนึ่ง user หนึ่งแถว)
IF OBJECT_ID('dbo.eval_assignment','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_assignment (
    id         INT IDENTITY(1,1) PRIMARY KEY,
    form_id    INT NOT NULL FOREIGN KEY REFERENCES dbo.eval_form(id) ON DELETE CASCADE,
    user_id    INT NOT NULL FOREIGN KEY REFERENCES dbo.users(id) ON DELETE CASCADE,
    status     TINYINT NOT NULL DEFAULT 0, -- 0=draft,1=submitted,2=approved (เผื่อใช้ภายหลัง)
    due_date   DATE NULL,
    created_at DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT uq_eval_assignment UNIQUE (form_id, user_id)
  );
END
GO

-- KPI ของ "ผู้ใช้" ในฟอร์ม (อิงตาม assignment)
IF OBJECT_ID('dbo.eval_kpi_user','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_kpi_user (
    id             INT IDENTITY(1,1) PRIMARY KEY,
    assignment_id  INT NOT NULL
      CONSTRAINT fk_kpi_user_assignment REFERENCES dbo.eval_assignment(id) ON DELETE CASCADE,
    idx            INT NOT NULL DEFAULT 0,            -- ลำดับ
    title          NVARCHAR(300) NOT NULL,            -- หัวข้อการประเมิน
    max_score      DECIMAL(6,2) NOT NULL DEFAULT 5.0, -- คะแนนเต็ม
    weight         INT NOT NULL DEFAULT 0,            -- ค่าถ่วงน้ำหนัก (%)
    expected_score DECIMAL(6,2) NULL,                 -- คะแนนความคาดหวัง
    measure        NVARCHAR(500) NULL,                -- วิธีการวัด
    criteria       NVARCHAR(1000) NULL,               -- เกณฑ์การให้คะแนน
    unit           NVARCHAR(50) NULL,                 -- หน่วย (ถ้ามี)
    code NVARCHAR(50) NULL,                -- รหัสหัวข้อการประเมิน
  score DECIMAL(6,2) NOT NULL DEFAULT 0, -- คะแนน (ตัวที่ผู้ใช้/ผู้ประเมินกรอก)
  note NVARCHAR(1000) NULL,              -- หมายเหตุ
    created_at     DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at     DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME()
  );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.check_constraints WHERE name='ck_eval_kpi_user_weight')
BEGIN
  ALTER TABLE dbo.eval_kpi_user
  ADD CONSTRAINT ck_eval_kpi_user_weight CHECK (weight BETWEEN 0 AND 100);
END
GO

-- ===== Competency items (shared by form; keep only header/score-related fields) =====
IF OBJECT_ID('dbo.eval_competency','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_competency (
    id             INT IDENTITY(1,1) PRIMARY KEY,
    form_id        INT NOT NULL
      CONSTRAINT fk_eval_comp_form REFERENCES dbo.eval_form(id) ON DELETE CASCADE,
    idx            INT NOT NULL DEFAULT 0,             -- ลำดับแสดงผล
    title          NVARCHAR(300) NOT NULL,             -- หัวข้อการประเมิน
    max_score      DECIMAL(10,2) NOT NULL DEFAULT 0,   -- คะแนนเต็มของหัวข้อ
    weight         DECIMAL(18,4) NOT NULL DEFAULT 0,   -- ค่าถ่วงน้ำหนัก (ส่งแบบ 0.70 หรือ 70 ก็ได้)
    full_total     DECIMAL(18,2) NOT NULL DEFAULT 0,   -- คะแนนเต็มรวม (ให้ UI คำนวณแล้วส่งมา)
    expected_score DECIMAL(10,2) NOT NULL DEFAULT 0,   -- คะแนนคาดหวัง
    created_at     DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at     DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME()
  );
END
GO


/* 1) คะแนน Competency ต่อผู้ใช้/ฟอร์ม (หัวข้ออยู่ที่ dbo.eval_competency) */
IF OBJECT_ID('dbo.eval_competency_score','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_competency_score (
  id            INT IDENTITY(1,1) PRIMARY KEY,
  assignment_id INT NOT NULL REFERENCES dbo.eval_assignment(id) ON DELETE CASCADE,
  comp_id       INT NOT NULL REFERENCES dbo.eval_competency(id),  -- ตัด CASCADE ออก
  score         DECIMAL(6,2) NOT NULL DEFAULT 0,
  note          NVARCHAR(1000) NULL,
  created_at    DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
  updated_at    DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
  CONSTRAINT uq_eval_comp_score UNIQUE (assignment_id, comp_id)
);
END
GO

/* 2) คะแนน Time Attendance (รวมเป็นชุดเดียว) */
IF OBJECT_ID('dbo.eval_ta_score','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_ta_score (
    assignment_id INT PRIMARY KEY
      REFERENCES dbo.eval_assignment(id) ON DELETE CASCADE,
    full_score    DECIMAL(6,2) NOT NULL DEFAULT 0,  -- คะแนนเต็ม
    score         DECIMAL(6,2) NOT NULL DEFAULT 0,  -- คะแนนได้
    updated_at    DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME()
  );
END
GO

/* 3) Development Plan (เป็นรายการ เพิ่มได้เท่าไหร่ก็เก็บเท่านั้น) */
IF OBJECT_ID('dbo.eval_dev_plan','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_dev_plan (
    id            INT IDENTITY(1,1) PRIMARY KEY,
    assignment_id INT NOT NULL REFERENCES dbo.eval_assignment(id) ON DELETE CASCADE,
    idx           INT NOT NULL DEFAULT 0,
    content       NVARCHAR(1000) NOT NULL,
    priority      NVARCHAR(20) NOT NULL DEFAULT 'High', -- High/Medium/Low (ปรับได้)
    timing        DATE NULL,
    remarks       NVARCHAR(500) NULL
  );
END
GO

/* 4) การประเมินเพิ่มเติม (5 ข้อ แบบข้อความ) */
IF OBJECT_ID('dbo.eval_additional','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_additional (
    assignment_id INT PRIMARY KEY
      REFERENCES dbo.eval_assignment(id) ON DELETE CASCADE,
    q1 NVARCHAR(2000) NULL,
    q2 NVARCHAR(2000) NULL,
    q3 NVARCHAR(2000) NULL,
    q4 NVARCHAR(2000) NULL,
    q5 NVARCHAR(2000) NULL,
    updated_at DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME()
  );
END
GO

IF OBJECT_ID('dbo.eval_grade_band','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_grade_band(
    id       INT IDENTITY(1,1) PRIMARY KEY,
    form_id  INT NULL,                -- NULL = default ทุกฟอร์ม
    grade    NVARCHAR(10) NOT NULL,   -- A, B+, C, D, F, FF ฯลฯ
    min_pct  DECIMAL(5,2) NOT NULL,   -- รวมเปอร์เซ็นต์ต่ำสุด (รวม)
    max_pct  DECIMAL(5,2) NOT NULL,   -- รวมเปอร์เซ็นต์สูงสุด (รวม)
    CONSTRAINT ck_band_range CHECK (min_pct <= max_pct)
  );
END
GO


-- ตารางเก็บขั้นตอนการประเมิน
IF OBJECT_ID('dbo.eval_step','U') IS NULL
BEGIN
  CREATE TABLE dbo.eval_step (
    id            INT IDENTITY(1,1) PRIMARY KEY,
    assignment_id INT NOT NULL FOREIGN KEY REFERENCES dbo.eval_assignment(id) ON DELETE CASCADE,
    idx           INT NOT NULL,                -- ลำดับขั้น (1=Self, 2, 3, ...)
    evaluator_id  INT NOT NULL FOREIGN KEY REFERENCES dbo.users(id),
    status        TINYINT NOT NULL DEFAULT 0,  -- 0=รอ,1=กำลังทำ,2=เสร็จ
    eval_date     DATE NULL,
    created_at    DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at    DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME()
  );
END
GO

